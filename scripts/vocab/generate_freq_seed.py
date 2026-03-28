from __future__ import annotations

import argparse
import csv
import sys
import sqlite3
from collections import Counter
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

from common import (  # noqa: E402
    CLOSED_CLASS_WORDS,
    DEFAULT_DB_PATH,
    DEFAULT_OUT_DIR,
    DEFAULT_WORDS_PATH,
    LEIPZIG_SOURCE_CORPUS,
    VALID_POS,
    WN_TO_POS,
    ensure_parent,
    is_simple_english_token,
    normalize_lemma,
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate freq_seed.csv from Leipzig + wnjpn.db")
    parser.add_argument("--db", type=Path, default=DEFAULT_DB_PATH)
    parser.add_argument("--words-file", type=Path, default=DEFAULT_WORDS_PATH)
    parser.add_argument("--out", type=Path, default=DEFAULT_OUT_DIR / "freq_seed.csv")
    parser.add_argument("--limit", type=int, default=0)
    parser.add_argument("--source-limit", type=int, default=0)
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


def iter_leipzig_rows(words_file: Path):
    with words_file.open("r", encoding="utf-8") as handle:
        for line_no, line in enumerate(handle, start=1):
            parts = line.rstrip("\n").split("\t")
            if len(parts) < 3:
                continue
            token = normalize_lemma(parts[1])
            try:
                frequency_count = int(parts[2])
            except ValueError as exc:
                raise ValueError(f"invalid frequency count at {words_file}:{line_no}") from exc
            yield token, frequency_count


def main() -> None:
    args = parse_args()
    ensure_parent(args.out)

    lemma_pos_map = load_lemma_pos_map(args.db)
    rank = 0
    rows: list[dict[str, object]] = []
    seen: set[tuple[str, str]] = set()
    seen_tokens: set[str] = set()
    pos_counts: Counter[str] = Counter()
    source_rows = 0

    for token, frequency_count in iter_leipzig_rows(args.words_file):
        if token in seen_tokens:
            continue
        seen_tokens.add(token)
        source_rows += 1
        if args.source_limit > 0 and source_rows > args.source_limit:
            break
        if token in CLOSED_CLASS_WORDS:
            continue
        if len(token) < args.min_length:
            continue
        if not is_simple_english_token(token):
            continue
        for pos in lemma_pos_map.get(token, []):
            key = (token, pos)
            if key in seen:
                continue
            seen.add(key)
            rank += 1
            rows.append(
                {
                    "lemma": token,
                    "pos": pos,
                    "frequency_rank": rank,
                    "frequency_count": frequency_count,
                    "source_token": token,
                    "source_corpus": LEIPZIG_SOURCE_CORPUS,
                }
            )
            pos_counts[pos] += 1
            if args.limit > 0 and rank >= args.limit:
                break
        if args.limit > 0 and rank >= args.limit:
            break

    with args.out.open("w", encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(
            handle,
            fieldnames=[
                "lemma",
                "pos",
                "frequency_rank",
                "frequency_count",
                "source_token",
                "source_corpus",
            ],
        )
        writer.writeheader()
        writer.writerows(rows)

    print(f"wrote {args.out}")
    print(f"rows={len(rows)}")
    print(f"source_tokens={len(seen_tokens)}")
    print("pos_counts=" + ", ".join(f"{pos}:{pos_counts[pos]}" for pos in sorted(pos_counts)))


if __name__ == "__main__":
    main()
