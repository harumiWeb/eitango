from __future__ import annotations

import argparse
import csv
import sqlite3
import sys
from collections import Counter
from pathlib import Path

from wordfreq import top_n_list

SCRIPT_DIR = Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

from common import (  # noqa: E402
    CLOSED_CLASS_WORDS,
    DEFAULT_DB_PATH,
    DEFAULT_OUT_DIR,
    VALID_POS,
    WN_TO_POS,
    ensure_parent,
    is_simple_english_token,
    normalize_lemma,
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate freq_seed.csv from wordfreq + wnjpn.db")
    parser.add_argument("--db", type=Path, default=DEFAULT_DB_PATH)
    parser.add_argument("--out", type=Path, default=DEFAULT_OUT_DIR / "freq_seed.csv")
    parser.add_argument("--limit", type=int, default=5000)
    parser.add_argument("--source-limit", type=int, default=20000)
    parser.add_argument("--min-length", type=int, default=4)
    return parser.parse_args()


def load_lemma_pos_map(db_path: Path) -> dict[str, list[str]]:
    conn = sqlite3.connect(db_path)
    rows = conn.execute(
        """
        SELECT DISTINCT lower(w.lemma) AS lemma, syn.pos
        FROM word w
        JOIN sense s
          ON s.wordid = w.wordid
         AND s.lang = 'eng'
        JOIN synset syn
          ON syn.synset = s.synset
        WHERE w.lang = 'eng'
          AND syn.pos IN ('n', 'v', 'a', 's', 'r')
        """
    ).fetchall()
    conn.close()

    pos_map: dict[str, list[str]] = {}
    for lemma, wn_pos in rows:
        mapped = WN_TO_POS.get(wn_pos)
        if mapped not in VALID_POS:
            continue
        bucket = pos_map.setdefault(lemma, [])
        if mapped not in bucket:
            bucket.append(mapped)
    return pos_map


def main() -> None:
    args = parse_args()
    ensure_parent(args.out)

    lemma_pos_map = load_lemma_pos_map(args.db)
    rank = 0
    rows: list[dict[str, object]] = []
    seen: set[tuple[str, str]] = set()
    pos_counts: Counter[str] = Counter()

    for token in top_n_list("en", args.source_limit):
        lemma = normalize_lemma(token)
        if lemma in CLOSED_CLASS_WORDS:
            continue
        if len(lemma) < args.min_length:
            continue
        if not is_simple_english_token(lemma):
            continue
        for pos in lemma_pos_map.get(lemma, []):
            key = (lemma, pos)
            if key in seen:
                continue
            seen.add(key)
            rank += 1
            rows.append(
                {
                    "lemma": lemma,
                    "pos": pos,
                    "frequency_rank": rank,
                }
            )
            pos_counts[pos] += 1
            if rank >= args.limit:
                break
        if rank >= args.limit:
            break

    with args.out.open("w", encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=["lemma", "pos", "frequency_rank"])
        writer.writeheader()
        writer.writerows(rows)

    print(f"wrote {args.out}")
    print(f"rows={len(rows)}")
    print("pos_counts=" + ", ".join(f"{pos}:{pos_counts[pos]}" for pos in sorted(pos_counts)))


if __name__ == "__main__":
    main()
