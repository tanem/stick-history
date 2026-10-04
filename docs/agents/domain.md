# Domain Docs

How the engineering skills should use this repo's domain documentation when exploring the codebase.

## Before exploring, read these

- **`GLOSSARY.md`** at the repo root, or
- **`GLOSSARY-MAP.md`** at the repo root if it exists. It points at one `GLOSSARY.md` per context. Read each one relevant to the topic.
- **`docs/adr/`**: read the ADRs that touch the area you are about to work in. In a multi-context repo, also check `src/<context>/docs/adr/` for decisions scoped to a context.

If any of these files do not exist, carry on without comment. Do not flag that they are missing, and do not suggest creating them in advance.

The `/domain-modeling` skill creates them when a term or a decision is resolved. It is reached through `/grill-with-docs` and `/improve-codebase-architecture`.

## File structure

This repo is single-context.

A single-context repo, which most repos are:

```
/
├── GLOSSARY.md
├── docs/adr/
│   ├── 0001-event-sourced-orders.md
│   └── 0002-postgres-for-write-model.md
└── src/
```

A multi-context repo, which has `GLOSSARY-MAP.md` at the root:

```
/
├── GLOSSARY-MAP.md
├── docs/adr/                          ← system-wide decisions
└── src/
    ├── ordering/
    │   ├── GLOSSARY.md
    │   └── docs/adr/                  ← context-specific decisions
    └── billing/
        ├── GLOSSARY.md
        └── docs/adr/
```

## Use the glossary's vocabulary

When your output names a domain concept, use the term as `GLOSSARY.md` defines it. This applies to an issue title, a refactor proposal, a hypothesis or a test name, for example. Do not use a synonym the glossary says to avoid.

If the concept is not in the glossary yet, one of two things is true:

- You are inventing language the project does not use. Reconsider it.
- The glossary has a gap. Note it for `/domain-modeling`.

## Flag ADR conflicts

If your output contradicts an existing ADR, say so. Do not override the ADR silently. For example:

> _Contradicts ADR-0007 (event-sourced orders), but worth reopening because…_
