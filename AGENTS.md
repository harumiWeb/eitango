# AI Agents Guide

## 0. Overview

This repository is organized around the following top-level directories:

```text
eitango/
├─ cmd/
│  └─ eitango/
│     └─ main.go
├─ internal/
│  ├─ app/          # Bubble Tea の state machine
│  ├─ tui/          # keymap, style, components
│  ├─ quiz/         # 出題ロジック
│  ├─ srs/          # 復習間隔計算
│  ├─ store/        # SQLite access
│  ├─ dict/         # 埋め込み辞書ロード
│  ├─ session/      # 学習セッション管理
│  ├─ stats/        # 統計集計
│  └─ config/       # パス/設定
├─ assets/
│  ├─ words_core.jsonl
│  └─ migrations/
│     ├─ 001_init.sql
│     └─ 002_indexes.sql
├─ .goreleaser.yaml
└─ go.mod
```

## 1. Workflow Design

### 1. Use Plan mode by default

- Always start tasks with 3 or more steps, or tasks that affect architecture, in Plan mode
- If things stop going well partway through, do not force it; stop immediately and replan
- Use Plan mode not only for implementation, but also for verification steps
- Write detailed specifications before implementation to reduce ambiguity

### 2. Multi-Agent Strategy

- Actively use sub-agents to keep the main context window clean
- Delegate research, investigation, and parallel analysis to sub-agents
- For complex problems, use sub-agents to apply more compute resources
- To keep execution focused, assign one task per sub-agent
- Use explorer for read-heavy codebase exploration
- Use worker for implementation and fixes
- Use reviewer for reviews

### 3. Self-Improvement Loop

- Whenever you receive a correction from the user, record that pattern in `tasks/lessons.md`
- Write rules for yourself so you do not repeat the same mistake
- Keep improving those rules thoroughly until the error rate goes down
- At the start of each session, review the lessons relevant to the project

### 4. Always verify before completion

- Do not mark a task as complete until you can prove that it works
- Compare the main branch and your changes when necessary
- Ask yourself, "Would a staff engineer approve this?"
- Run tests, review logs, and show that it works correctly

### 5. Pursue elegance (with balance)

- Before making an important change, pause and ask, "Is there a more elegant way to do this?"
- If a fix feels hacky, think, "Based on everything I know now, implement an elegant solution"
- Skip this process for simple and obvious fixes (do not over-engineer)
- Question your own work before presenting it

### 6. Autonomous bug fixing

- When you receive a bug report, fix it directly without needing step-by-step guidance
- Use logs, errors, and failing tests to solve it yourself
- Eliminate context switching for the user
- Even without being asked, go fix failing CI tests

---

## 2. Documentation Retention Policy

### Separation of Roles

- `tasks/todo.md` may temporarily hold not only session-specific progress tracking, but also verification results, unresolved items, and summaries of decision rationale.
- `tasks/feature_spec.md` may be used as a pre-implementation working spec draft, but do not treat it as disposable if it contains specifications, constraints, or validation conditions that will be referenced in the future.
- `tasks/lessons.md` is where recurrence-prevention rules are stored, and should not be used to store design decisions or the specification itself.
- Move design decisions and trade-offs to `docs/adr/`, current internal specifications and constraints to `docs/specs/`.

### Information to Keep

- Decision rationale that future implementers may encounter again on the same issue
- Chosen policies adopted after comparing multiple options
- Permanent rules established through review, CI, or incident response
- Contracts related to CLI, validation, and compatibility
- Specification context behind added regression tests where forgetting the reason could cause the issue to recur

### Information You May Discard

- One-off notes about work order
- Rejected hypotheses or interim notes that ended midway
- Progress logs with no reference value after completion
- Simple lists of steps with no decision rationale

---

## 3. Core Principles

- **Simplicity first**: Keep every change as simple as possible. Minimize the code affected.
- **No cutting corners**: Find the root cause. Avoid temporary fixes. Maintain senior engineer standards.
- **Minimize impact**: Limit changes to only what is necessary. Do not introduce new bugs.
