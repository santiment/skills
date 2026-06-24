# Skills

Agent-facing `SKILL.md` files, one directory per skill. Each skill documents a
CLI tool (or workflow) so an AI agent knows when and how to use it.

## Layout

```
skills/
├── _template/SKILL.md   # copy this to start a new skill
├── score-cli/SKILL.md   # skill for the `score` CLI
└── <name>/SKILL.md      # one folder per skill
```

## Adding a skill

1. Copy `_template/SKILL.md` to `skills/<name>/SKILL.md`.
2. Fill the YAML frontmatter: `name` (kebab-case) and a `description` of the form
   *verb + object + "Use when <trigger>"* — readable on its own so an agent can
   decide relevance from the description alone.
3. Complete every section. For CLI tools (utility skills) the **TOOL USAGE**,
   **OUTPUT REQUIREMENTS**, and **FAILURE HANDLING** sections are the core:
   give exact commands, the output contract, and concrete recovery steps.
4. Keep it a procedure (SOP), not a tutorial. Prefer 1–2 GOOD/BAD example pairs
   over long prose.

## Conventions

- Document stable exit codes and `--json` output for any CLI skill.
- Mark state-changing operations explicitly in DECISION RULES.
- Keep the skill self-contained; do not rely on external docs to be usable.
