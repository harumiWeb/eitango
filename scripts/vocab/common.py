from __future__ import annotations

import csv
import json
import re
from pathlib import Path
from typing import Iterable


SCRIPT_DIR = Path(__file__).resolve().parent
REPO_ROOT = SCRIPT_DIR.parent.parent
DEFAULT_DB_PATH = REPO_ROOT / "tmp" / "wnjpn.db"
DEFAULT_WORDS_PATH = REPO_ROOT / "tmp" / "eng_news_2024_1M-words.txt"
DEFAULT_CORE_PATH = REPO_ROOT / "assets" / "words_core.jsonl"
DEFAULT_OUT_DIR = REPO_ROOT / "tmp" / "generated_vocab"
LEIPZIG_SOURCE_CORPUS = "leipzig:eng_news_2024_1M"

POS_TO_WN = {
    "noun": ("n",),
    "verb": ("v",),
    "adjective": ("a", "s"),
    "adverb": ("r",),
}

WN_TO_POS = {
    "n": "noun",
    "v": "verb",
    "a": "adjective",
    "s": "adjective",
    "r": "adverb",
}

VALID_POS = ("noun", "verb", "adjective", "adverb")
REVIEW_TSV_FIELDS = [
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
]
WORD_RE = re.compile(r"^[a-z]+(?:-[a-z]+)?$")
TOKEN_RE = re.compile(r"[A-Za-z]+(?:'[A-Za-z]+)?")
ASCII_RE = re.compile(r"[A-Za-z]")

# A compact blocklist is enough here because the pipeline only targets content-word
# expansion for the current core schema.
CLOSED_CLASS_WORDS = {
    "a",
    "about",
    "above",
    "across",
    "after",
    "against",
    "along",
    "also",
    "although",
    "an",
    "and",
    "another",
    "any",
    "anyone",
    "anything",
    "anywhere",
    "are",
    "around",
    "as",
    "at",
    "be",
    "because",
    "before",
    "behind",
    "below",
    "been",
    "being",
    "between",
    "beyond",
    "both",
    "but",
    "by",
    "can",
    "could",
    "did",
    "do",
    "does",
    "down",
    "during",
    "each",
    "either",
    "else",
    "enough",
    "every",
    "everyone",
    "everything",
    "few",
    "for",
    "from",
    "had",
    "has",
    "have",
    "he",
    "her",
    "hers",
    "him",
    "his",
    "i",
    "if",
    "in",
    "inside",
    "into",
    "is",
    "it",
    "its",
    "itself",
    "just",
    "may",
    "many",
    "might",
    "more",
    "most",
    "much",
    "must",
    "my",
    "myself",
    "near",
    "neither",
    "no",
    "none",
    "nor",
    "not",
    "nothing",
    "of",
    "off",
    "on",
    "once",
    "only",
    "onto",
    "or",
    "other",
    "otherwise",
    "our",
    "ours",
    "ourselves",
    "out",
    "outside",
    "over",
    "own",
    "same",
    "several",
    "she",
    "should",
    "since",
    "some",
    "someone",
    "something",
    "sometimes",
    "somewhere",
    "such",
    "than",
    "that",
    "the",
    "their",
    "theirs",
    "them",
    "themselves",
    "these",
    "they",
    "this",
    "those",
    "through",
    "throughout",
    "to",
    "too",
    "toward",
    "under",
    "until",
    "up",
    "upon",
    "us",
    "via",
    "was",
    "we",
    "were",
    "what",
    "when",
    "where",
    "whether",
    "which",
    "while",
    "who",
    "whom",
    "whose",
    "why",
    "will",
    "with",
    "within",
    "without",
    "would",
    "you",
    "your",
    "yours",
    "yourself",
    "yourselves",
}

VERB_GROUP_KEYWORDS = {
    "communication-verb": (
        "answer",
        "announce",
        "argue",
        "ask",
        "call",
        "chat",
        "claim",
        "communicate",
        "complain",
        "confess",
        "describe",
        "discuss",
        "explain",
        "inform",
        "mention",
        "post",
        "present",
        "protest",
        "publish",
        "question",
        "read",
        "reply",
        "report",
        "say",
        "shout",
        "speak",
        "state",
        "suggest",
        "talk",
        "tell",
        "text",
        "translate",
        "tweet",
        "warn",
        "write",
    ),
    "thinking-verb": (
        "assume",
        "believe",
        "choose",
        "compare",
        "consider",
        "deduce",
        "decide",
        "doubt",
        "estimate",
        "expect",
        "feel",
        "forget",
        "guess",
        "imagine",
        "judge",
        "know",
        "learn",
        "memorize",
        "notice",
        "plan",
        "prefer",
        "prove",
        "realize",
        "reason",
        "recall",
        "recognize",
        "remember",
        "suppose",
        "think",
        "understand",
        "wonder",
    ),
    "business-verb": (
        "apply",
        "book",
        "borrow",
        "budget",
        "buy",
        "cancel",
        "charge",
        "collect",
        "contract",
        "cost",
        "deliver",
        "earn",
        "employ",
        "exchange",
        "export",
        "finance",
        "hire",
        "import",
        "invest",
        "lend",
        "manage",
        "market",
        "order",
        "package",
        "pay",
        "postpone",
        "profit",
        "purchase",
        "refund",
        "register",
        "rent",
        "reserve",
        "sell",
        "ship",
        "sign",
        "supply",
        "trade",
    ),
}

NOUN_GROUP_KEYWORDS = {
    "people-noun": (
        "actor",
        "adult",
        "agent",
        "boss",
        "boy",
        "child",
        "citizen",
        "client",
        "colleague",
        "consumer",
        "couple",
        "crowd",
        "customer",
        "employee",
        "engineer",
        "family",
        "friend",
        "girl",
        "guest",
        "human",
        "manager",
        "member",
        "neighbor",
        "nurse",
        "officer",
        "owner",
        "passenger",
        "patient",
        "person",
        "player",
        "president",
        "relative",
        "resident",
        "staff",
        "student",
        "teacher",
        "tourist",
        "user",
        "visitor",
        "worker",
    ),
    "place-noun": (
        "airport",
        "apartment",
        "area",
        "bank",
        "beach",
        "building",
        "campus",
        "center",
        "city",
        "classroom",
        "clinic",
        "club",
        "college",
        "corridor",
        "country",
        "factory",
        "farm",
        "hall",
        "home",
        "hospital",
        "hotel",
        "kitchen",
        "lab",
        "location",
        "market",
        "museum",
        "office",
        "park",
        "restaurant",
        "room",
        "school",
        "shop",
        "site",
        "station",
        "store",
        "town",
        "university",
        "village",
        "zone",
    ),
    "travel-noun": (
        "airline",
        "bicycle",
        "boat",
        "bus",
        "cab",
        "car",
        "flight",
        "ferry",
        "gas",
        "gate",
        "guide",
        "journey",
        "map",
        "passport",
        "path",
        "platform",
        "rail",
        "road",
        "route",
        "seat",
        "ship",
        "station",
        "subway",
        "taxi",
        "ticket",
        "tour",
        "train",
        "transport",
        "travel",
        "trip",
        "vehicle",
    ),
    "business-noun": (
        "account",
        "budget",
        "business",
        "cash",
        "client",
        "company",
        "contract",
        "cost",
        "credit",
        "customer",
        "deal",
        "debt",
        "demand",
        "discount",
        "economy",
        "expense",
        "fee",
        "finance",
        "fund",
        "industry",
        "insurance",
        "interest",
        "inventory",
        "investment",
        "invoice",
        "market",
        "meeting",
        "money",
        "office",
        "order",
        "payment",
        "price",
        "product",
        "profit",
        "project",
        "receipt",
        "revenue",
        "salary",
        "sale",
        "service",
        "share",
        "staff",
        "stock",
        "supply",
        "tax",
        "trade",
    ),
    "technology-noun": (
        "app",
        "camera",
        "cell",
        "chip",
        "code",
        "computer",
        "data",
        "device",
        "display",
        "engine",
        "file",
        "image",
        "keyboard",
        "machine",
        "memory",
        "network",
        "phone",
        "program",
        "robot",
        "screen",
        "sensor",
        "server",
        "signal",
        "software",
        "system",
        "tablet",
        "technology",
        "tool",
        "video",
        "web",
    ),
    "learning-noun": (
        "article",
        "book",
        "class",
        "course",
        "culture",
        "dictionary",
        "education",
        "essay",
        "grammar",
        "history",
        "idea",
        "knowledge",
        "language",
        "lesson",
        "library",
        "meaning",
        "math",
        "note",
        "paper",
        "poem",
        "question",
        "research",
        "science",
        "skill",
        "story",
        "subject",
        "study",
        "text",
        "theory",
        "vocabulary",
        "word",
    ),
}

ADJECTIVE_GROUP_KEYWORDS = {
    "emotion-adjective": (
        "afraid",
        "angry",
        "anxious",
        "ashamed",
        "calm",
        "cheerful",
        "confident",
        "curious",
        "depressed",
        "eager",
        "embarrassed",
        "excited",
        "glad",
        "gloomy",
        "grateful",
        "happy",
        "jealous",
        "lonely",
        "nervous",
        "proud",
        "relaxed",
        "sad",
        "scared",
        "serious",
        "shy",
        "surprised",
        "tired",
        "uneasy",
        "upset",
        "worried",
    ),
    "business-adjective": (
        "available",
        "commercial",
        "competitive",
        "corporate",
        "economic",
        "effective",
        "efficient",
        "financial",
        "formal",
        "global",
        "industrial",
        "legal",
        "local",
        "official",
        "organizational",
        "political",
        "practical",
        "professional",
        "public",
        "technical",
    ),
    "condition-adjective": (
        "alive",
        "asleep",
        "awake",
        "broken",
        "clean",
        "closed",
        "cold",
        "dry",
        "empty",
        "full",
        "healthy",
        "hot",
        "hungry",
        "ill",
        "loose",
        "open",
        "prepared",
        "ready",
        "safe",
        "sick",
        "sleepy",
        "tired",
        "wet",
    ),
}

TIME_ADVERBS = {
    "already",
    "before",
    "early",
    "eventually",
    "finally",
    "immediately",
    "late",
    "later",
    "meanwhile",
    "now",
    "recently",
    "soon",
    "still",
    "suddenly",
    "then",
    "today",
    "tomorrow",
    "tonight",
    "yesterday",
}

FREQUENCY_ADVERBS = {
    "again",
    "always",
    "constantly",
    "daily",
    "frequently",
    "generally",
    "hardly",
    "never",
    "normally",
    "occasionally",
    "often",
    "rarely",
    "regularly",
    "repeatedly",
    "seldom",
    "sometimes",
    "usually",
}


def ensure_parent(path: Path) -> None:
    path.parent.mkdir(parents=True, exist_ok=True)


def is_simple_english_token(value: str) -> bool:
    return bool(WORD_RE.fullmatch(value))


def normalize_lemma(value: str) -> str:
    return value.strip().lower()


def word_count(text: str) -> int:
    return len(TOKEN_RE.findall(text))


def contains_ascii(text: str) -> bool:
    return bool(ASCII_RE.search(text))


def load_jsonl(path: Path) -> list[dict]:
    rows: list[dict] = []
    if not path.exists():
        return rows
    with path.open("r", encoding="utf-8") as handle:
        for line in handle:
            if line.strip():
                rows.append(json.loads(line))
    return rows


def write_jsonl(path: Path, rows: Iterable[dict]) -> None:
    ensure_parent(path)
    with path.open("w", encoding="utf-8", newline="") as handle:
        for row in rows:
            handle.write(json.dumps(row, ensure_ascii=False) + "\n")


def load_tsv(path: Path, expected_fieldnames: list[str] | None = None) -> list[dict[str, str]]:
    if not path.exists():
        return []
    with path.open("r", encoding="utf-8-sig", newline="") as handle:
        reader = csv.reader(handle, delimiter="\t")
        try:
            header = next(reader)
        except StopIteration:
            return []
        if len(header) != len(set(header)):
            raise ValueError(f"duplicate TSV columns in {path}: {header}")
        if expected_fieldnames is not None and header != expected_fieldnames:
            raise ValueError(f"unexpected TSV header in {path}: {header}")

        rows: list[dict[str, str]] = []
        for line_no, row in enumerate(reader, start=2):
            if not row or all(cell == "" for cell in row):
                continue
            if len(row) != len(header):
                raise ValueError(
                    f"row {line_no} in {path} has {len(row)} columns; expected {len(header)}"
                )
            rows.append({name: row[index] for index, name in enumerate(header)})
        return rows


def write_tsv(path: Path, fieldnames: list[str], rows: Iterable[dict[str, object]]) -> None:
    ensure_parent(path)
    with path.open("w", encoding="utf-8", newline="") as handle:
        writer = csv.DictWriter(handle, fieldnames=fieldnames, delimiter="\t")
        writer.writeheader()
        for row in rows:
            writer.writerow({name: row.get(name, "") for name in fieldnames})


def load_core_entries(core_path: Path) -> list[dict]:
    return load_jsonl(core_path)


def load_core_keys(core_path: Path) -> set[tuple[str, str]]:
    keys = set()
    for row in load_core_entries(core_path):
        keys.add((normalize_lemma(row["lemma"]), row["pos"]))
    return keys


def looks_reasonable_japanese(text: str) -> bool:
    value = text.strip()
    if not value:
        return False
    if contains_ascii(value):
        return False
    if len(value) > 12:
        return False
    if any(ch in value for ch in ("(", ")", "[", "]", "{", "}", ";", ":", "・")):
        return False
    if value.endswith(("こと", "もの", "ため")) and len(value) > 4:
        return False
    return True


def simple_variants(lemma: str, pos: str) -> set[str]:
    variants = {lemma}
    if pos == "noun":
        if lemma.endswith("y") and len(lemma) > 2:
            variants.add(lemma[:-1] + "ies")
        elif lemma.endswith(("s", "x", "z", "ch", "sh")):
            variants.add(lemma + "es")
        else:
            variants.add(lemma + "s")
    elif pos == "verb":
        if lemma.endswith("e"):
            variants.add(lemma + "d")
            variants.add(lemma[:-1] + "ing")
        elif lemma.endswith("y") and len(lemma) > 2:
            variants.add(lemma[:-1] + "ied")
            variants.add(lemma + "ing")
        else:
            variants.add(lemma + "ed")
            variants.add(lemma + "ing")
        if lemma.endswith(("s", "x", "z", "ch", "sh", "o")):
            variants.add(lemma + "es")
        else:
            variants.add(lemma + "s")
    elif pos in {"adjective", "adverb"}:
        variants.add(lemma + "ly")
    return variants


def sentence_mentions_lemma(sentence: str, lemma: str, pos: str) -> bool:
    lowered = sentence.lower()
    return any(re.search(rf"\b{re.escape(variant)}\b", lowered) for variant in simple_variants(lemma, pos))


def core_level_for_index(index: int, total: int) -> str:
    if total <= 0:
        raise ValueError("total must be positive")
    if index < 0 or index >= total:
        raise ValueError(f"index {index} out of range for total {total}")
    bucket = min((index * 4) // total, 3)
    return f"core-{bucket + 1}"


def suggest_level(source_rank: int, total_ranks: int) -> str:
    if source_rank <= 0:
        raise ValueError("source_rank must be positive")
    return core_level_for_index(source_rank - 1, total_ranks)


def _has_keyword(lemma: str, text: str, keywords: tuple[str, ...]) -> bool:
    haystack = f"{lemma} {text}".lower()
    return any(keyword in haystack for keyword in keywords)


def suggest_group(lemma: str, pos: str, text: str) -> str:
    if pos == "verb":
        for group, keywords in VERB_GROUP_KEYWORDS.items():
            if _has_keyword(lemma, text, keywords):
                return group
        return "basic-verb-action"
    if pos == "noun":
        for group_name in (
            "people-noun",
            "place-noun",
            "travel-noun",
            "business-noun",
            "technology-noun",
            "learning-noun",
        ):
            if _has_keyword(lemma, text, NOUN_GROUP_KEYWORDS[group_name]):
                return group_name
        return "daily-noun"
    if pos == "adjective":
        for group_name in ("emotion-adjective", "business-adjective", "condition-adjective"):
            if _has_keyword(lemma, text, ADJECTIVE_GROUP_KEYWORDS[group_name]):
                return group_name
        return "quality-adjective"
    if pos == "adverb":
        lowered = f"{lemma} {text}".lower()
        if any(keyword in lowered for keyword in TIME_ADVERBS):
            return "time-adverb"
        if any(keyword in lowered for keyword in FREQUENCY_ADVERBS):
            return "frequency-adverb"
        return "manner-adverb"
    raise ValueError(f"unsupported pos: {pos}")

