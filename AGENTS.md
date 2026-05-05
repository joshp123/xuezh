# AGENTS.md — Project operating rules (read before touching code)

This is the root AGENTS.md per the agentsmd.io convention. Subdirectories may
add their own AGENTS.md when needed.

You are implementing `xuezh`, a local Chinese learning engine used behind a Telegram bot + SOTA LLM.

Start here:
- Documentation authority map: `docs/README.md`
- Repo overview + usage: `README.md`

Authoritative surfaces (must stay in sync):
1) **CLI contract (human)**: `docs/cli-contract.md`
2) **CLI contract (machine)**: `specs/cli/contract.json`
3) **Output schemas**: `schemas/`
4) **Executable BDD specs**: `specs/bdd/`
5) **Contract sync tests**: `tests/contract/`

High-signal supporting specs (treat as binding):
- ZFC boundary + invariants: `specs/invariants.md`, `docs/architecture.md`
- North stars + URs: `specs/north-stars.md`, `specs/user-requirements.md`
- IDs, events, retention: `specs/id-scheme.md`, `specs/events.md`, `specs/artifacts/retention.md`
- Audio backend policy: `specs/audio-backends.md`
- HSK scope: `specs/hsk-scope.md`
- Skill reference (consumer contract): `skills/chinese-learning-orchestrator/SKILL.md`

Issue tracking:
- Beads: use `bd` (see `bd --help` / `bd onboard`)

First steps for a new agent:
1) `bd ready` → pick a ticket
2) `bd show <id>` → read acceptance + constraints
3) `direnv allow` → load `devenv`
4) `devenv shell` → enter project env
5) `./scripts/check.sh` → verify baseline

Repo layout (what lives where):
- `src/xuezh/`: core engine + CLI entry points
- `specs/`: specs, contracts, and executable BDD feature files
- `schemas/`: JSON schemas for CLI command outputs
- `tests/`: unit + contract sync tests
- `skills/`: LLM skill docs + examples (consumer contract)
- `tickets/`: implementation tickets (source of work)
- `docs/`: architecture + contract docs + reference material

Working a ticket (minimum steps):
1) Read `tickets/<id>.md` and confirm North Star + UR mapping
2) Verify contract impacts against `docs/README.md` authority map
3) Update tests first (RGR), then code, then docs
4) Run `./scripts/check.sh`
5) Update ticket notes and close it

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

   Quick setup:
   - `direnv allow`
   - `devenv shell`

2) **ZFC compliance**
   - Engine must never do:
     - ranking/scoring/selection heuristics
     - “what to learn next” recommendations
     - scheduling policy or lesson planning
   - Engine may do:
     - store primary sources
     - bounded reports (facts)
     - mechanical transforms (audio conversion, schedule rule application when explicitly chosen)

   Concrete guardrails (see `specs/invariants.md`):
   - Do not add “best next lesson” logic or scoring/ranking logic.
   - Any selection must be externally provided or explicitly specified by the user.

3) **Single-user system**
   - Do not implement multi-user features.
   - No `--user` flags. Workspace represents the single learner.

4) **CLI contract is authoritative**
   - `docs/cli-contract.md` is the single source of truth.
   - `specs/cli/contract.json` is the machine-readable source of truth.
   - `schemas/` must match actual outputs.
   - BDD features must match the CLI and schemas.

   Contract change checklist:
   - Update `docs/cli-contract.md` + `specs/cli/contract.json`
   - Update `schemas/<command>.schema.json`
   - Update/add `specs/bdd/*.feature`
   - Update contract sync tests in `tests/contract/` if needed

   Contract drift will fail `tests/contract/` — fix the contract, schema, and BDD in the same ticket.

5) **Pre-release schema policy**
   - This repo is pre-release. Prefer editing the current schema directly over
     migration chains, compatibility layers, fallback reads, dual writes, or
     defensive shims.
   - If local dev data matters, make a one-off backup/export first, then keep
     the application code simple.

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

Non-human commits:
- Any automated/agent commit must set an explicit author identifying the agent
  (e.g., `Codex <codex@openai.com>`), not a human identity.
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

## Plugin configuration (nix-openclaw)

Environment knobs (set via plugin `config.env`):
- `XUEZH_AZURE_SPEECH_KEY_FILE`: path to the Azure Speech key file (e.g. `/run/agenix/xuezh-azure-speech-key`)
- `XUEZH_AZURE_SPEECH_REGION`: Azure region (e.g. `westeurope`)

Config resolution:
- Default path: `~/.config/xuezh/config.toml` (or `$XDG_CONFIG_HOME/xuezh/config.toml`)

## Cram app operating notes

- Local runner: `scripts/run-cram-local.sh`; default URL is `http://127.0.0.1:8765/`.
- Canonical content comes from the shared HelloChinese/Pleco text source plus Travel Survival text. Pleco backup content is only for matching score/recency metadata, not for replacing card text.
- Audio generation is fill-missing by default. Existing voice files are reused when the DB path exists and the file is present. Do not regenerate/replace audio unless Josh explicitly asks; replacement requires `XUEZH_AUDIO_REPLACE=1` / `--replace`.
- The calibrated default voices/rates are in `scripts/run-cram-local.sh`: Xiaoxiao `-23%`, Xiaoyi `-15%`, Yunxi `-15%`, Yunyang `-25%`.
- For UI-only debugging, start with `XUEZH_SKIP_AUDIO_BACKFILL=1 ./scripts/run-cram-local.sh` so review work cannot accidentally kick off TTS.
- The web server sets no-store headers for app assets; if the in-app browser looks stale, force reload before judging the UI.
- Design-system route: `http://127.0.0.1:8765/#design-system`. It must include type roles, controls, learning bars, flashcard before/after/long states, and history.
- Design-system implementation belongs under `web/src/dev/` and must be lazy-loaded from the hash route. Do not import dev/design-system CSS from `web/src/styles.css` or production app components.
- The design system must render the same production React components used by the app for important surfaces, especially the review card. Do not create fake duplicate flashcard markup/CSS that can drift from production.
- The design-system page has in-browser regression checks. They should pass before handoff; if they fail, fix the component/CSS instead of explaining the failure away.
- Generic design-system single-card routes such as `#design-system-long-answer` must test desktop-responsive layout. Phone-only routes must say `phone` in the hash, e.g. `#design-system-phone-long-answer`.
- Phone-only CSS overrides in the design system must be scoped to phone specimen classes. Desktop specimens should use the normal production component layout.
- Active review rounds are server-owned. The browser may cache UI-only state, but the authoritative current card, remaining queue, retry queue, reveal state, and reviewed history live in `cram_review_sessions`. Do not reintroduce `sessionStorage` as the source of truth for review progress.
- Review answers must be ACID: score row, review event, and review-session queue update happen together, or none of them happen. Undo must restore both the previous score row and the previous session queue snapshot.
- Mobile QA matters. Check at about iPhone width (`393x852` is fine): no horizontal overflow, no truncated action labels, pinyin only after reveal, source headers sticky in the review-picker list, and visible rows not bleeding through sticky headers.
- For any web UI change, run one proactive visual pass before handoff. Build/restart, then capture screenshots for at least:
  - review card before reveal
  - review card after reveal
  - review card with long Chinese sentence
  - review card with long English answer
  - picker with sticky category headers
- Screenshots must use the actual target CSS viewport, not a cropped desktop viewport. If using headless Chrome, drive DevTools `Emulation.setDeviceMetricsOverride` to `393x852`; `--window-size=393,852` alone can still lay out at 500px and hide mobile regressions.
- Reject the UI if screenshots show wrapped header/footer controls, controls covering text, clipped text, text hidden behind mobile browser chrome, horizontal overflow, unstable prompt position between reveal states, or production and design-system components disagreeing.
- Do not split Chinese prompt text into one span per character. It breaks copy/paste by inserting line breaks between every Hanzi. Keep sentence text as normal text nodes except the highlighted target span.
- Do grading/session smoke tests in a throwaway workspace, not Josh's real `~/.local/share/xuezh/cram-local` DB. Use `XUEZH_WORKSPACE_DIR=/private/tmp/xuezh-cram-smoke-...` and import with `--audio none`; start the local web server on a spare port and verify start → reveal → incorrect → reload active session → correct → undo. Only use the real DB for read-only UI screenshots unless Josh explicitly asks to mutate it.
