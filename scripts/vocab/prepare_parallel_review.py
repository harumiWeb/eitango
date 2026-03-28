from __future__ import annotations

import argparse
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
    parser = argparse.ArgumentParser(description="Split review candidates into parallel review slice TSVs")
    parser.add_argument("--review-tsv", type=Path, default=DEFAULT_OUT_DIR / "review_candidates.tsv")
    parser.add_argument("--approved-tsv", type=Path, default=DEFAULT_OUT_DIR / "approved_review_candidates.tsv")
    parser.add_argument("--out-dir", type=Path, default=DEFAULT_OUT_DIR / "parallel_review")
    parser.add_argument("--min-rank", type=int, required=True)
    parser.add_argument("--max-rank", type=int, required=True)
    parser.add_argument("--slice-size", type=int, default=100)
    return parser.parse_args()


def load_approved_keys(path: Path) -> set[tuple[str, str]]:
    keys: set[tuple[str, str]] = set()
    for row in load_tsv(path, expected_fieldnames=REVIEW_TSV_FIELDS):
        if row["status"] != "approved":
            continue
        keys.add((normalize_lemma(row["lemma"]), row["pos"]))
    return keys


def main() -> None:
    args = parse_args()
    if args.min_rank > args.max_rank:
        raise ValueError("--min-rank must be <= --max-rank")
    if args.slice_size <= 0:
        raise ValueError("--slice-size must be positive")

    approved_keys = load_approved_keys(args.approved_tsv)
    candidates = []
    for row in load_tsv(args.review_tsv, expected_fieldnames=REVIEW_TSV_FIELDS):
        rank = int(row["source_frequency_rank"])
        if rank < args.min_rank or rank > args.max_rank:
            continue
        key = (normalize_lemma(row["lemma"]), row["pos"])
        if key in approved_keys:
            continue
        candidates.append(row)

    candidates.sort(key=lambda row: (int(row["source_frequency_rank"]), row["pos"], row["lemma"]))

    if not candidates:
        print("candidates=0")
        return

    args.out_dir.mkdir(parents=True, exist_ok=True)
    for path in args.out_dir.glob("review_slice_*.tsv"):
        path.unlink()
    for path in args.out_dir.glob("approved_slice_*.tsv"):
        path.unlink()

    slice_count = 0
    for slice_count, start in enumerate(range(0, len(candidates), args.slice_size), start=1):
        chunk = candidates[start : start + args.slice_size]
        out_path = args.out_dir / f"review_slice_{slice_count:02d}.tsv"
        write_tsv(out_path, REVIEW_TSV_FIELDS, chunk)
        print(
            f"slice={slice_count:02d} rows={len(chunk)} "
            f"ranks={chunk[0]['source_frequency_rank']}-{chunk[-1]['source_frequency_rank']} "
            f"path={out_path}"
        )

    print(f"candidates={len(candidates)}")
    print(f"slices={slice_count}")


if __name__ == "__main__":
    main()
