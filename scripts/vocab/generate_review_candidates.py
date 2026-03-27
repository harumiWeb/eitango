from __future__ import annotations

import argparse
import csv
import sys
from pathlib import Path

SCRIPT_DIR = Path(__file__).resolve().parent
if str(SCRIPT_DIR) not in sys.path:
    sys.path.insert(0, str(SCRIPT_DIR))

from common import (  # noqa: E402
    DEFAULT_CORE_PATH,
    DEFAULT_OUT_DIR,
    load_core_keys,
    load_jsonl,
    load_tsv,
    looks_reasonable_japanese,
    suggest_group,
    suggest_level,
    write_tsv,
)


BAD_DEFINITION_KEYWORDS = (
    "a state in",
    "state in new england",
    "roman letter",
    "the smallest whole number",
    "numeral representing",
    "unit of",
    "chemical element",
    "radioactive",
    "halogen",
    "a single person or thing",
    "indefinite in time or position",
)


def parse_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description="Generate review_candidates.tsv from intermediate artifacts")
    parser.add_argument("--core", type=Path, default=DEFAULT_CORE_PATH)
    parser.add_argument("--freq-seed", type=Path, default=DEFAULT_OUT_DIR / "freq_seed.csv")
    parser.add_argument("--meaning-candidates", type=Path, default=DEFAULT_OUT_DIR / "meaning_candidates.jsonl")
    parser.add_argument("--examples", type=Path, default=DEFAULT_OUT_DIR / "examples.tsv")
    parser.add_argument("--out", type=Path, default=DEFAULT_OUT_DIR / "review_candidates.tsv")
    parser.add_argument("--min-score", type=int, default=8)
    parser.add_argument("--min-lemma-length", type=int, default=4)
    return parser.parse_args()


def build_meaning_index(path: Path) -> dict[tuple[str, str], dict]:
    rows = {}
    for row in load_jsonl(path):
        rows[(row["lemma"], row["pos"])] = row
    return rows


def build_example_index(path: Path) -> dict[tuple[str, str], dict[str, str]]:
    rows: dict[tuple[str, str], dict[str, str]] = {}
    for row in load_tsv(path):
        key = (row["lemma"], row["pos"])
        if key not in rows:
            rows[key] = row
    return rows


def main() -> None:
    args = parse_args()
    core_keys = load_core_keys(args.core)
    meaning_index = build_meaning_index(args.meaning_candidates)
    example_index = build_example_index(args.examples)
    review_rows: list[dict[str, object]] = []

    with args.freq_seed.open("r", encoding="utf-8", newline="") as handle:
        reader = csv.DictReader(handle)
        for seed_row in reader:
            key = (seed_row["lemma"], seed_row["pos"])
            if key in core_keys:
                continue
            if len(seed_row["lemma"]) < args.min_lemma_length:
                continue
            meaning_row = meaning_index.get(key)
            if not meaning_row or not meaning_row["candidates"]:
                continue
            best = meaning_row["candidates"][0]
            if int(best["score_hint"]) < args.min_score:
                continue
            if not looks_reasonable_japanese(best["ja"]):
                continue
            alternatives = [candidate["ja"] for candidate in meaning_row["candidates"][1:4]]
            text_for_group = " ".join(best.get("en_defs", [])[:3] + best.get("ja_defs", [])[:2])
            lowered_text = text_for_group.lower()
            if any(keyword in lowered_text for keyword in BAD_DEFINITION_KEYWORDS):
                continue
            example = example_index.get(key, {})
            review_rows.append(
                {
                    "status": "candidate",
                    "lemma": seed_row["lemma"],
                    "pos": seed_row["pos"],
                    "source_frequency_rank": seed_row["frequency_rank"],
                    "meaning_ja_candidate": best["ja"],
                    "meaning_ja_alternatives": " / ".join(alternatives),
                    "level_candidate": suggest_level(int(seed_row["frequency_rank"])),
                    "distractor_group_candidate": suggest_group(seed_row["lemma"], seed_row["pos"], text_for_group),
                    "example_en": example.get("example_en", ""),
                    "example_ja": example.get("example_ja", ""),
                    "confidence": best["score_hint"],
                    "source_synsets": ",".join(best.get("synsets", [])[:6]),
                    "notes": "; ".join(best.get("ja_defs", [])[:2] + best.get("en_defs", [])[:1]),
                }
            )

    review_rows.sort(
        key=lambda item: (
            int(item["source_frequency_rank"]),
            item["pos"],
            item["lemma"],
        )
    )

    write_tsv(
        args.out,
        [
            "status",
            "lemma",
            "pos",
            "source_frequency_rank",
            "meaning_ja_candidate",
            "meaning_ja_alternatives",
            "level_candidate",
            "distractor_group_candidate",
            "example_en",
            "example_ja",
            "confidence",
            "source_synsets",
            "notes",
        ],
        review_rows,
    )
    print(f"wrote {args.out}")
    print(f"rows={len(review_rows)}")


if __name__ == "__main__":
    main()

