from __future__ import annotations

import argparse
import csv
import json
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

from common import (  # noqa: E402
    DEFAULT_CORE_PATH,
    DEFAULT_OUT_DIR,
    core_level_for_index,
    load_core_entries,
    normalize_lemma,
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(
        description="Rebuild words_core.jsonl by retaining only current core rows backed by freq_seed.csv"
    )
    parser.add_argument("--core", type=Path, default=DEFAULT_CORE_PATH)
    parser.add_argument("--freq-seed", type=Path, default=DEFAULT_OUT_DIR / "freq_seed.csv")
    parser.add_argument("--out", type=Path, default=DEFAULT_CORE_PATH)
    parser.add_argument("--report", type=Path, default=DEFAULT_OUT_DIR / "core_rebuild_report.tsv")
    return parser.parse_args()


def load_seed_index(path: Path) -> dict[tuple[str, str], dict[str, str]]:
    with path.open("r", encoding="utf-8", newline="") as handle:
        reader = csv.DictReader(handle)
        return {
            (normalize_lemma(row["lemma"]), row["pos"]): row
            for row in reader
        }


def main() -> None:
    args = parse_args()
    core_entries = load_core_entries(args.core)
    seed_index = load_seed_index(args.freq_seed)

    retained: list[dict] = []
    dropped: list[dict] = []

    for entry in core_entries:
        key = (normalize_lemma(entry["lemma"]), entry["pos"])
        seed_row = seed_index.get(key)
        if seed_row is None:
            dropped.append(
                {
                    "status": "dropped",
                    "lemma": key[0],
                    "pos": entry["pos"],
                    "old_frequency_rank": entry.get("frequency_rank", ""),
                    "new_frequency_rank": "",
                    "old_level": entry.get("level", ""),
                    "new_level": "",
                    "seed_frequency_rank": "",
                    "frequency_count": "",
                    "source_token": "",
                    "source_corpus": "",
                    "reason": "missing from Leipzig+WordNet freq seed",
                }
            )
            continue

        updated = dict(entry)
        updated["lemma"] = key[0]
        updated["_seed_frequency_rank"] = int(seed_row["frequency_rank"])
        updated["_frequency_count"] = seed_row.get("frequency_count", "")
        updated["_source_token"] = seed_row.get("source_token", "")
        updated["_source_corpus"] = seed_row.get("source_corpus", "")
        retained.append(updated)

    retained.sort(key=lambda row: (int(row["_seed_frequency_rank"]), row["pos"], row["lemma"]))

    report_rows: list[dict[str, object]] = []
    for index, entry in enumerate(retained):
        new_rank = index + 1
        new_level = core_level_for_index(index, len(retained))
        report_rows.append(
            {
                "status": "retained",
                "lemma": entry["lemma"],
                "pos": entry["pos"],
                "old_frequency_rank": entry.get("frequency_rank", ""),
                "new_frequency_rank": new_rank,
                "old_level": entry.get("level", ""),
                "new_level": new_level,
                "seed_frequency_rank": entry["_seed_frequency_rank"],
                "frequency_count": entry["_frequency_count"],
                "source_token": entry["_source_token"],
                "source_corpus": entry["_source_corpus"],
                "reason": "",
            }
        )
        entry["frequency_rank"] = new_rank
        entry["level"] = new_level
        del entry["_seed_frequency_rank"]
        del entry["_frequency_count"]
        del entry["_source_token"]
        del entry["_source_corpus"]

    report_rows.extend(dropped)
    report_rows.sort(key=lambda row: (row["status"] != "retained", str(row["seed_frequency_rank"] or "999999"), row["pos"], row["lemma"]))

    with args.out.open("w", encoding="utf-8", newline="") as handle:
        for row in retained:
            handle.write(json.dumps(row, ensure_ascii=False, separators=(",", ":")) + "\n")

    with args.report.open("w", encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(
            handle,
            fieldnames=[
                "status",
                "lemma",
                "pos",
                "old_frequency_rank",
                "new_frequency_rank",
                "old_level",
                "new_level",
                "seed_frequency_rank",
                "frequency_count",
                "source_token",
                "source_corpus",
                "reason",
            ],
            delimiter="\t",
        )
        writer.writeheader()
        writer.writerows(report_rows)

    print(f"retained={len(retained)}")
    print(f"dropped={len(dropped)}")
    print(f"wrote {args.out}")
    print(f"wrote {args.report}")


if __name__ == "__main__":
    main()
