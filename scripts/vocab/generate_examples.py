from __future__ import annotations

import argparse
import csv
import sqlite3
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

from common import (  # noqa: E402
    DEFAULT_DB_PATH,
    DEFAULT_OUT_DIR,
    normalize_lemma,
    sentence_mentions_lemma,
    word_count,
    write_tsv,
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Generate examples.tsv from wnjpn.db synset_ex pairs. Tatoeba can be layered later."
    )
    parser.add_argument("--db", type=Path, default=DEFAULT_DB_PATH)
    parser.add_argument("--freq-seed", type=Path, default=DEFAULT_OUT_DIR / "freq_seed.csv")
    parser.add_argument("--out", type=Path, default=DEFAULT_OUT_DIR / "examples.tsv")
    parser.add_argument("--max-words", type=int, default=12)
    parser.add_argument("--max-per-lemma", type=int, default=3)
    return parser.parse_args()


def load_seed_rows(path: Path) -> list[tuple[str, str, int]]:
    rows: list[tuple[str, str, int]] = []
    with path.open("r", encoding="utf-8", newline="") as handle:
        reader = csv.DictReader(handle)
        for row in reader:
            rows.append((normalize_lemma(row["lemma"]), row["pos"], int(row["frequency_rank"])))
    return rows


def main() -> None:
    args = parse_args()
    conn = sqlite3.connect(args.db)
    seed_rows = load_seed_rows(args.freq_seed)

    conn.execute("DROP TABLE IF EXISTS temp.seed_vocab")
    conn.execute(
        """
        CREATE TEMP TABLE seed_vocab (
            lemma TEXT NOT NULL,
            pos TEXT NOT NULL,
            frequency_rank INTEGER NOT NULL
        )
        """
    )
    conn.executemany(
        "INSERT INTO seed_vocab (lemma, pos, frequency_rank) VALUES (?, ?, ?)",
        seed_rows,
    )
    conn.execute("CREATE INDEX temp.seed_vocab_idx ON seed_vocab (lemma, pos)")

    query = """
        SELECT DISTINCT
            seed.lemma,
            seed.pos,
            s.synset,
            ex_eng.sid,
            ex_eng.def AS example_en,
            ex_jpn.def AS example_ja,
            COALESCE(CAST(s.freq AS INTEGER), 0) AS eng_freq
        FROM seed_vocab seed
        JOIN word w
          ON w.lang = 'eng'
         AND lower(w.lemma) = seed.lemma
        JOIN sense s
          ON s.wordid = w.wordid
         AND s.lang = 'eng'
        JOIN synset syn
          ON syn.synset = s.synset
        JOIN synset_ex ex_eng
          ON ex_eng.synset = s.synset
         AND ex_eng.lang = 'eng'
        JOIN synset_ex ex_jpn
          ON ex_jpn.synset = ex_eng.synset
         AND ex_jpn.sid = ex_eng.sid
         AND ex_jpn.lang = 'jpn'
        WHERE (seed.pos = 'noun' AND syn.pos = 'n')
           OR (seed.pos = 'verb' AND syn.pos = 'v')
           OR (seed.pos = 'adjective' AND syn.pos IN ('a', 's'))
           OR (seed.pos = 'adverb' AND syn.pos = 'r')
    """

    grouped_rows: dict[tuple[str, str], list[dict[str, object]]] = {}
    for lemma, pos, synset, sid, example_en, example_ja, eng_freq in conn.execute(query):
        if not sentence_mentions_lemma(example_en, lemma, pos):
            continue
        if word_count(example_en) > args.max_words:
            continue
        grouped_rows.setdefault((lemma, pos), []).append(
            {
                "lemma": lemma,
                "pos": pos,
                "synset": synset,
                "sid": sid,
                "source": "wnjpn:synset_ex",
                "example_en": example_en.strip(),
                "example_ja": example_ja.strip(),
                "eng_freq": int(eng_freq),
            }
        )

    rows: list[dict[str, object]] = []
    for items in grouped_rows.values():
        items.sort(key=lambda item: (word_count(str(item["example_en"])), -int(item["eng_freq"]), str(item["example_en"])))
        rows.extend(items[: args.max_per_lemma])

    conn.close()

    write_tsv(
        args.out,
        ["lemma", "pos", "synset", "sid", "source", "example_en", "example_ja"],
        rows,
    )
    print(f"wrote {args.out}")
    print(f"rows={len(rows)}")


if __name__ == "__main__":
    main()
