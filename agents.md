# agents.md — Implementation rules (read before touching code)

You are implementing `xuezh`, a local Chinese learning engine used behind a Telegram bot + SOTA LLM.

## North stars (must be referenced in every ticket)

- **NS-1**: Improve Chinese with low effort via short, enjoyable, frequent sessions (natural acquisition, i+1, speaking).
- **NS-2**: Provide auditable instrumentation (HSK coverage/recency, modality splits) as **facts**.
- **NS-3**: ZFC/Unix boundary: engine is a thin deterministic shell; model is the pedagogy.

See `specs/north-stars.md`.

## Non-negotiable constraints

1) **Use `devenv`**
   - Do not use brew/global installs.
   - System tools belong in `devenv.nix`.
   - Python deps belong in `pyproject.toml` and are installed in the devenv venv.

2) **ZFC compliance**
   - Engine must never do:
     - ranking/scoring/selection heuristics
     - “what to learn next” recommendations
     - scheduling policy or lesson planning
   - Engine may do:
     - store primary sources
     - bounded reports (facts)
     - mechanical transforms (audio conversion, schedule rule application when explicitly chosen)

3) **Single-user system**
   - Do not implement multi-user features.
   - No `--user` flags. Workspace represents the single learner.

4) **CLI contract is authoritative**
   - `docs/cli-contract.md` is the single source of truth.
   - `specs/cli/contract.json` is the machine-readable source of truth.
   - `schemas/` must match actual outputs.
   - BDD features must match the CLI and schemas.

## RGR workflow (required)

For each ticket:
- **Red**: enable/extend tests to express acceptance criteria.
- **Green**: implement the smallest change to pass.
- **Refactor**: clean up, keep tests green.

## Traceability checklist (every ticket must satisfy)

When completing a ticket, update:
1) The ticket file: mark status, add notes on decisions.
2) **Tests**:
   - unit / integration / e2e coverage per `specs/test-strategy.md`
   - if behavior is user-visible, add/adjust BDD scenario(s)
3) **Contract artifacts** if affected:
   - `docs/cli-contract.md`
   - `specs/cli/contract.json`
   - `schemas/`
   - `skills/.../SKILL.md` (should link, not duplicate)
4) **North star & UR mapping**:
   - ensure ticket still maps to URs and NS.
   - do not introduce “nice-to-have” features not tied to URs.

## Testing pyramid enforcement

- Most logic must be unit-testable.
- CLI contract tests must validate JSON envelope + schema.
- BDD suite is executable (pytest-bdd). While commands are NOT_IMPLEMENTED, scenarios xfail.
  After implementation, scenarios become strict automatically.

See `specs/test-strategy.md`.


## Git & GitHub discipline (required)

Before starting tickets:
1) Initialize git repo:
   ```bash
   git init
   git add .
   git commit -m "chore: initial scaffold"
   ```

2) Create a **private** GitHub repo using `gh` and push:
   ```bash
   gh auth status
   gh repo create <YOUR_REPO_NAME> --private --source . --remote origin --push
   ```

### One ticket = one atomic commit
- For every ticket:
  - implement using RGR
  - run `./scripts/check.sh`
  - commit **once** with message: `T-XX: <ticket title>`
  - **no fixup commits**, no “WIP” commits
- If you made intermediate commits locally, you must squash into a single commit before pushing.

### Pushing
- Push after each ticket:
  ```bash
  git push
  ```


## Stop-the-line gates

Some tickets are marked `requires_user_review: true` in YAML frontmatter.
For those tickets:
- produce the requested artifact/sample output
- stop and request user approval
- do not proceed to dependent tickets until approval is recorded in the ticket notes


## Contract coverage policy (enforced by tests)

Every CLI command listed in `specs/cli/contract.json` MUST have:
- a command-specific JSON schema at `schemas/<command_id>.schema.json`
- at least one executable BDD scenario under `specs/bdd/` that invokes it
- a ticket mapping in `contract.json` (`ticket: T-XX`)
- and the mapped ticket must declare `implements_commands: [...]` including that command id

The contract tests will fail if any of these drift.

## Landing the Plane (Session Completion)

**When ending a work session**, you MUST complete ALL steps below. Work is NOT complete until `git push` succeeds.

**MANDATORY WORKFLOW:**

1. **File issues for remaining work** - Create issues for anything that needs follow-up
2. **Run quality gates** (if code changed) - Tests, linters, builds
3. **Update issue status** - Close finished work, update in-progress items
4. **PUSH TO REMOTE** - This is MANDATORY:
   ```bash
   git pull --rebase
   bd sync
   git push
   git status  # MUST show "up to date with origin"
   ```
5. **Clean up** - Clear stashes, prune remote branches
6. **Verify** - All changes committed AND pushed
7. **Hand off** - Provide context for next session

**CRITICAL RULES:**
- Work is NOT complete until `git push` succeeds
- NEVER stop before pushing - that leaves work stranded locally
- NEVER say "ready to push when you are" - YOU must push
- If push fails, resolve and retry until it succeeds
