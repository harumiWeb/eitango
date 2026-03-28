from __future__ import annotations

import argparse
import json
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

from common import (  # noqa: E402
    DEFAULT_CORE_PATH,
    REVIEW_TSV_FIELDS,
    load_tsv,
    load_core_entries,
    normalize_lemma,
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Apply approved rows from review_candidates.tsv into words_core.jsonl")
    parser.add_argument("review_tsv", type=Path)
    parser.add_argument("--core", type=Path, default=DEFAULT_CORE_PATH)
    parser.add_argument("--out", type=Path, default=None)
    parser.add_argument("--status", default="approved")
    return parser.parse_args()


def main() -> None:
    args = parse_args()
    core_entries = load_core_entries(args.core)
    seen = {(normalize_lemma(row["lemma"]), row["pos"]) for row in core_entries}
    next_rank = max(int(row["frequency_rank"]) for row in core_entries)
    out_path = args.out or args.core

    approved_rows = []
    for row in load_tsv(args.review_tsv, expected_fieldnames=REVIEW_TSV_FIELDS):
        if row["status"] != args.status:
            continue
        if row.get("confidence"):
            int(row["confidence"])
        if row.get("example_ja", "").strip().isdigit():
            raise ValueError(f"numeric example_ja is not allowed for {row['lemma']}:{row['pos']}")
        key = (normalize_lemma(row["lemma"]), row["pos"])
        if key in seen:
            continue
        approved_rows.append(row)
        seen.add(key)

    approved_rows.sort(key=lambda row: int(row["source_frequency_rank"]))

    for row in approved_rows:
        next_rank += 15
        entry = {
            "lemma": normalize_lemma(row["lemma"]),
            "pos": row["pos"],
            "meaning_ja": row["meaning_ja_candidate"],
            "level": row["level_candidate"],
            "frequency_rank": next_rank,
            "distractor_group": row["distractor_group_candidate"],
        }
        if row.get("example_en"):
            entry["example_en"] = row["example_en"]
        if row.get("example_ja"):
            entry["example_ja"] = row["example_ja"]
        core_entries.append(entry)

    with out_path.open("w", encoding="utf-8", newline="") as handle:
        for row in core_entries:
            handle.write(json.dumps(row, ensure_ascii=False, separators=(",", ":")) + "\n")

    print(f"applied={len(approved_rows)}")
    print(f"wrote {out_path}")


if __name__ == "__main__":
    main()
