from __future__ import annotations

import argparse
import csv
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

from common import (  # noqa: E402
    DEFAULT_OUT_DIR,
    REVIEW_TSV_FIELDS,
    load_tsv,
    normalize_lemma,
    write_tsv,
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Merge approved parallel review slice TSVs")
    parser.add_argument("--base-approved-tsv", type=Path, default=DEFAULT_OUT_DIR / "approved_review_candidates.tsv")
    parser.add_argument("--slice-dir", type=Path, required=True)
    parser.add_argument("--slice-glob", default="approved_slice_*.tsv")
    parser.add_argument("--out-approved-tsv", type=Path, default=DEFAULT_OUT_DIR / "approved_review_candidates.tsv")
    parser.add_argument("--out-seed-csv", type=Path, default=DEFAULT_OUT_DIR / "approved_seed.csv")
    return parser.parse_args()


def validate_review_row(row: dict[str, str], source: str) -> None:
    if row["status"] != "approved":
        raise ValueError(f"{source} has non-approved status: {row['status']!r}")
    if not normalize_lemma(row["lemma"]):
        raise ValueError(f"{source} has empty lemma")
    if not row["pos"]:
        raise ValueError(f"{source} has empty pos")
    if not row["meaning_ja_candidate"]:
        raise ValueError(f"{source} has empty meaning_ja_candidate")
    if not row["level_candidate"]:
        raise ValueError(f"{source} has empty level_candidate")
    if not row["distractor_group_candidate"]:
        raise ValueError(f"{source} has empty distractor_group_candidate")
    int(row["source_frequency_rank"])
    if row["confidence"]:
        int(row["confidence"])
    if row["example_ja"].strip().isdigit():
        raise ValueError(f"{source} has numeric-only example_ja: {row['example_ja']!r}")


def load_approved_rows(path: Path) -> list[dict[str, str]]:
    rows = load_tsv(path, expected_fieldnames=REVIEW_TSV_FIELDS)
    for index, row in enumerate(rows, start=2):
        validate_review_row(row, f"{path}:{index}")
    return rows


def main() -> None:
    args = parse_args()
    base_rows = load_approved_rows(args.base_approved_tsv)
    merged_rows = list(base_rows)
    seen = {(normalize_lemma(row["lemma"]), row["pos"]) for row in base_rows}

    slice_paths = sorted(args.slice_dir.glob(args.slice_glob))
    if not slice_paths:
        raise ValueError(f"no slice files matched {args.slice_glob} in {args.slice_dir}")

    new_rows: list[dict[str, str]] = []
    for path in slice_paths:
        for index, row in enumerate(load_tsv(path, expected_fieldnames=REVIEW_TSV_FIELDS), start=2):
            validate_review_row(row, f"{path}:{index}")
            key = (normalize_lemma(row["lemma"]), row["pos"])
            if key in seen:
                raise ValueError(f"duplicate approved key from slice merge: {key}")
            seen.add(key)
            new_rows.append(row)

    merged_rows.extend(new_rows)
    merged_rows.sort(key=lambda row: (int(row["source_frequency_rank"]), row["pos"], normalize_lemma(row["lemma"])))

    write_tsv(args.out_approved_tsv, REVIEW_TSV_FIELDS, merged_rows)
    with args.out_seed_csv.open("w", encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=["lemma", "pos", "frequency_rank"])
        writer.writeheader()
        for row in merged_rows:
            writer.writerow(
                {
                    "lemma": normalize_lemma(row["lemma"]),
                    "pos": row["pos"],
                    "frequency_rank": row["source_frequency_rank"],
                }
            )

    print(f"merged_base={len(base_rows)}")
    print(f"merged_new={len(new_rows)}")
    print(f"merged_total={len(merged_rows)}")
    print(f"wrote {args.out_approved_tsv}")
    print(f"wrote {args.out_seed_csv}")


if __name__ == "__main__":
    main()
