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
    contains_ascii,
    looks_reasonable_japanese,
    normalize_lemma,
    write_jsonl,
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate meaning_candidates.jsonl from wnjpn.db")
    parser.add_argument("--db", type=Path, default=DEFAULT_DB_PATH)
    parser.add_argument("--freq-seed", type=Path, default=DEFAULT_OUT_DIR / "freq_seed.csv")
    parser.add_argument("--out", type=Path, default=DEFAULT_OUT_DIR / "meaning_candidates.jsonl")
    parser.add_argument("--max-candidates", type=int, default=12)
    return parser.parse_args()


def load_seed_rows(path: Path) -> list[tuple[str, str, int]]:
    rows: list[tuple[str, str, int]] = []
    with path.open("r", encoding="utf-8", newline="") as handle:
        reader = csv.DictReader(handle)
        for row in reader:
            rows.append((normalize_lemma(row["lemma"]), row["pos"], int(row["frequency_rank"])))
    return rows


def score_candidate(lemma: str, ja: str, ja_def: str, eng_def: str, eng_freq: int, jpn_freq: int) -> int:
    score = min(eng_freq, 6) * 3 + min(jpn_freq, 6) * 2
    if looks_reasonable_japanese(ja):
        score += 8
    if len(ja) <= 4:
        score += 3
    elif len(ja) <= 8:
        score += 2
    elif len(ja) > 12:
        score -= 6
    if contains_ascii(ja):
        score -= 8
    if ja == lemma:
        score -= 10
    if "、" in ja or "。" in ja:
        score -= 4
    if any(ch in ja for ch in ("(", ")", "[", "]", "{", "}", ";", ":", "・")):
        score -= 3
    if any(ch in ja for ch in ("ゐ", "ゑ", "ヰ", "ヱ")):
        score -= 5
    if ja_def and len(ja_def) <= 36:
        score += 1
    if eng_def and len(eng_def) <= 96:
        score += 1
    return score


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
        SELECT
            seed.lemma,
            seed.pos,
            seed.frequency_rank,
            s.synset,
            COALESCE(CAST(s.freq AS INTEGER), 0) AS eng_freq,
            COALESCE(CAST(sj.freq AS INTEGER), 0) AS jpn_freq,
            j.lemma AS ja,
            COALESCE(jd.def, '') AS ja_def,
            COALESCE(ed.def, '') AS eng_def
        FROM seed_vocab seed
        JOIN word w
          ON w.lang = 'eng'
         AND lower(w.lemma) = seed.lemma
        JOIN sense s
          ON s.wordid = w.wordid
         AND s.lang = 'eng'
        JOIN synset syn
          ON syn.synset = s.synset
        JOIN sense sj
          ON sj.synset = s.synset
         AND sj.lang = 'jpn'
        JOIN word j
          ON j.wordid = sj.wordid
         AND j.lang = 'jpn'
        LEFT JOIN synset_def jd
          ON jd.synset = s.synset
         AND jd.lang = 'jpn'
        LEFT JOIN synset_def ed
          ON ed.synset = s.synset
         AND ed.lang = 'eng'
        WHERE (seed.pos = 'noun' AND syn.pos = 'n')
           OR (seed.pos = 'verb' AND syn.pos = 'v')
           OR (seed.pos = 'adjective' AND syn.pos IN ('a', 's'))
           OR (seed.pos = 'adverb' AND syn.pos = 'r')
    """

    grouped_rows: dict[tuple[str, str, int], dict[str, dict]] = {}
    for lemma, pos, frequency_rank, synset, eng_freq, jpn_freq, ja, ja_def, eng_def in conn.execute(query):
        key = (lemma, pos, int(frequency_rank))
        per_word = grouped_rows.setdefault(key, {})
        score = score_candidate(lemma, ja, ja_def, eng_def, int(eng_freq), int(jpn_freq))
        current = per_word.setdefault(
            ja,
            {
                "ja": ja,
                "score_hint": score,
                "synsets": [],
                "ja_defs": [],
                "en_defs": [],
            },
        )
        current["score_hint"] = max(int(current["score_hint"]), score)
        if synset not in current["synsets"]:
            current["synsets"].append(synset)
        if ja_def and ja_def not in current["ja_defs"]:
            current["ja_defs"].append(ja_def)
        if eng_def and eng_def not in current["en_defs"]:
            current["en_defs"].append(eng_def)

    rows = []
    for (lemma, pos, frequency_rank), candidates_by_ja in sorted(grouped_rows.items(), key=lambda item: item[0][2]):
        candidates = list(candidates_by_ja.values())
        candidates.sort(
            key=lambda item: (
                -int(item["score_hint"]),
                len(str(item["ja"])),
                str(item["ja"]),
            )
        )
        rows.append(
            {
                "lemma": lemma,
                "pos": pos,
                "frequency_rank": frequency_rank,
                "candidates": candidates[: args.max_candidates],
            }
        )

    conn.close()
    write_jsonl(args.out, rows)
    print(f"wrote {args.out}")
    print(f"rows={len(rows)}")


if __name__ == "__main__":
    main()
