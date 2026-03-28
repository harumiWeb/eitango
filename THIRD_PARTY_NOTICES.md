# Third-Party Notices

This repository contains project code, bundled vocabulary data, and development tooling with different provenance.

## 1. Project Code

- The `eitango` application code is licensed under Apache License 2.0.
- See [LICENSE](LICENSE) for the project license text.

## 2. Runtime Dependencies Distributed with the App

The Go application is built from modules that use permissive upstream licenses. The main direct dependencies currently referenced in `go.mod` are:

| Component | Role | Upstream license | Reference |
| --- | --- | --- | --- |
| `charm.land/bubbletea/v2` | TUI runtime | MIT | `third_party/licenses/MIT.txt` |
| `charm.land/bubbles/v2` | TUI components | MIT | `third_party/licenses/MIT.txt` |
| `charm.land/lipgloss/v2` | terminal styling | MIT | `third_party/licenses/MIT.txt` |
| `github.com/BurntSushi/toml` | TOML parsing | MIT | `third_party/licenses/MIT.txt` |
| `github.com/spf13/cobra` | CLI framework | Apache-2.0 | `third_party/licenses/Apache-2.0.txt` |
| `github.com/google/uuid` | UUID generation | BSD-3-Clause style | `third_party/licenses/BSD-3-Clause.txt` |
| `modernc.org/sqlite` | pure Go SQLite driver | BSD-style | `third_party/licenses/BSD-3-Clause.txt` |

These dependencies are part of the source distribution and linked binary builds. Retain the accompanying license texts when redistributing release artifacts.

## 3. Repository-Distributed Tooling for Vocabulary Generation

The repository also ships Python tooling under `scripts/vocab/` and declares the following Python package dependencies in `pyproject.toml`:

| Component | Role | Upstream license | Notes |
| --- | --- | --- | --- |
| `wordfreq` | frequency-based vocabulary seed generation | Apache-2.0 for code | upstream data lineage includes additional terms; see section 4 |
| `nltk` | NLP helper library | Apache-2.0 | runtime app does not require NLTK |

These tools are for repository maintenance and are not required for normal end-user execution of the compiled CLI.

## 4. Bundled Vocabulary Data and Upstream Data Lineage

The file `assets/words_core.jsonl` is bundled with the repository and seeded into the local database at runtime. It is a project-curated vocabulary file, but the current repository treats it conservatively as an edited dataset produced with help from upstream lexical resources.

Relevant upstream references:

| Upstream source | How it is used here | License reference |
| --- | --- | --- |
| Japanese WordNet (`wnjpn.db`) | queried by scripts under `scripts/vocab/` for meanings and examples | `third_party/licenses/Japanese-WordNet.txt` |
| `wordfreq` data sources | consulted for frequency-ranked seed generation | `third_party/licenses/CC-BY-SA-4.0.txt` |
| Princeton WordNet / WordNet-family notices | relevant to WordNet-derived corpora and notices | `third_party/licenses/Princeton-WordNet.txt` |

Important distribution note:

- this repository does **not** ship raw `wnjpn.db`
- this repository does **not** ship NLTK corpora as standalone data files
- this repository does **not** ship raw `wordfreq` datasets separately from the Python package
- this repository **does** ship `assets/words_core.jsonl`, which should be redistributed together with this notice and the referenced upstream license materials

If you redistribute `words_core.jsonl` or a derivative package that embeds it, keep this notice and review the referenced upstream terms before making additional licensing claims.

## 5. Practical Guidance for Redistributors

- Treat the application code license and the bundled data provenance as separate concerns.
- Do not describe the entire repository as if every bundled artifact were covered only by Apache-2.0.
- Include `LICENSE`, this file, and `third_party/licenses/` in source or binary redistributions.
- When in doubt about data provenance, keep the attribution and upstream links intact rather than removing them.
