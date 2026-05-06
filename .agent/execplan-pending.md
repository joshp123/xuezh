# Centralize OpenTofu Infrastructure and Retire Stale Product Stacks

This ExecPlan is a living document. The sections `Progress`, `Surprises & Discoveries`, `Decision Log`, and `Outcomes & Retrospective` must be kept up to date as work proceeds.

This repository has `.agent/PLANS.md`; maintain this document according to that file.

## Purpose / Big Picture

After this work, Josh has one obvious place for cloud resources managed with OpenTofu. AWS, Azure, Hetzner, GCP, DNS, and similar cloud/provider resources live in a single repo with a tree that explains itself. Application repos keep application code, content, and deploy scripts. Nix repos keep machine/runtime/service configuration. Secrets stay in the secrets repo. Agents can discover this without guessing because the global AGENTS preamble and affected repo AGENTS files point to the same source of truth.

The first practical migration is `jjpcodes.com` infrastructure, because it is active OpenTofu work and belongs in the central OpenTofu repo. Task Rally and OpenClaw Cloud are retired or superseded product stacks; they must be inventoried and reported before any destroy happens. Azure Speech for xuezh can become the first new resource in the central repo once the repo shape is agreed, but it should not force a bad ontology.

This plan does not destroy anything and does not move state until Josh approves the inventory and migration gates. It creates the operating model, inventories existing OpenTofu roots, migrates active website infrastructure first, and produces explicit decommission reports for retired stacks.

## Progress

- [x] (2026-05-06T15:20Z) Inspected current known OpenTofu roots across `/Users/josh/code`, including `website`, `taskrally/apps/cloud`, `openclaw-cloud`, `xuezh`, `nix/nixos-config`, and several smaller repos.
- [x] (2026-05-06T15:20Z) Confirmed `xuezh` worktree was clean before writing this plan.
- [x] (2026-05-06T15:20Z) Moved the completed whole-deck offline ExecPlan to `.agent/done/whole-deck-offline.md`.
- [x] (2026-05-06T15:20Z) Wrote this infra centralization ExecPlan for alignment.
- [ ] Align with Josh on repo name, top-level ontology, and state backend.
- [ ] Create or choose the central OpenTofu repo.
- [ ] Build read-only inventory reports for active, candidate, and retired stacks.
- [ ] Migrate `jjpcodes.com` OpenTofu into the central repo with a no-drift plan.
- [ ] Move xuezh Azure Speech OpenTofu scaffold into the central repo after the repo shape is proven.
- [ ] Produce Task Rally and OpenClaw Cloud destroy reports, then stop for approval.
- [ ] Update global and affected AGENTS docs so future agents know where OpenTofu belongs.

## Surprises & Discoveries

- Observation: `website` currently owns public site content and also some OpenTofu roots.
  Evidence: `website/AGENTS.md` describes `sites/gallery`, `sites/jjpcodes-blog`, and `sites/jjpcodes-home`; `website/infra/tofu/jjpcodes-blog` and `website/infra/tofu/eastwindlegal-nl` contain OpenTofu stacks.

- Observation: Task Rally and OpenClaw Cloud have many OpenTofu roots and may not share one state layout.
  Evidence: `taskrally/apps/cloud/infra/opentofu/envs/*` and `openclaw-cloud/infra/opentofu/envs/*` include roots for control planes, Hetzner hosts, jumpboxes, org bootstrap, Gmail auth, tenant DNS, and tenant cells.

- Observation: `nix/nixos-config` should not be the cloud-resource owner.
  Evidence: its AGENTS file frames it as host/system config and private runtime choices. It should consume cloud outputs or secrets, not own AWS/Azure/Hetzner resources directly.

- Observation: `xuezh` has an Azure Speech OpenTofu scaffold, but that belongs in the central OpenTofu repo once the new repo exists.
  Evidence: `xuezh/infra/azure/speech` contains a validated OpenTofu root for the xuezh speech account.

- Observation: There are OpenTofu/Terraform roots scattered in smaller repos too.
  Evidence: read-only file discovery found roots under `clawdinators`, `djtbot`, `gohome`, `lawbot`, `trmnl-inkhub`, `website`, `taskrally`, `openclaw-cloud`, and `xuezh`.

## Decision Log

- Decision: Use `opentofu-infra` as the working repo name in this plan.
  Rationale: Josh rejected `personal infra`; `opentofu-infra` is blunt, agent-discoverable, and names the actual boundary: OpenTofu-managed cloud resources. If Josh wants a less tool-named repo later, `cloud-infra` is the only serious alternative.
  Date/Author: 2026-05-06 / Codex

- Decision: Organize stacks by purpose first, provider second.
  Rationale: A human running `tree -L 3` should see `jjpcodes`, `xuezh`, `taskrally`, and `openclaw-cloud`, not a provider taxonomy that hides why resources exist. Provider names still appear in stack names and provider files.
  Date/Author: 2026-05-06 / Codex

- Decision: Retired product stacks get inventory and destroy reports before any destroy.
  Rationale: Task Rally and OpenClaw Cloud may have separate states, shared DNS, auth resources, or remnants already partly removed. Destroying from the wrong root or wrong state would be risky.
  Date/Author: 2026-05-06 / Codex

- Decision: Nix remains machine/runtime ownership; OpenTofu owns cloud/provider resources.
  Rationale: NixOS/Darwin configs should configure services, launchd, Caddy, Tailscale, environment files, and secret consumption. OpenTofu should own DNS zones, hosted zones, CDN, object buckets, cloud speech resources, cloud VMs, and provider-side IAM.
  Date/Author: 2026-05-06 / Codex

## Outcomes & Retrospective

Not started. Expected outcome is an approved repo layout, a committed central infra repo skeleton, a migrated no-drift `jjpcodes.com` stack, xuezh Azure Speech moved out of the app repo, and two written decommission reports that Josh can approve or reject before any destructive action.

## Context and Orientation

Current source repos and likely responsibilities:

- `/Users/josh/code/website`: public site source and deploy scripts. It currently contains OpenTofu roots for `jjpcodes` and `eastwindlegal-nl`, which should move.
- `/Users/josh/code/xuezh`: local Chinese learning app. It currently contains `infra/azure/speech`, which should move after the central repo exists.
- `/Users/josh/code/taskrally/apps/cloud`: retired/superseded product stack with many OpenTofu roots. Inventory first, destroy only after approval.
- `/Users/josh/code/openclaw-cloud`: older OpenClaw Cloud stack with separate OpenTofu roots. Inventory first, destroy only after approval.
- `/Users/josh/code/nix/nixos-config`: host/runtime/service config. It should reference cloud outputs and secrets, not own cloud resources.
- `/Users/josh/code/nix/nix-secrets`: secret material. Do not move secrets into OpenTofu.
- `/Users/josh/code/nix/ai-stack` and `/Users/josh/AGENTS.md`: likely places for global agent instructions. Locate the canonical deployed preamble before editing.

Proposed central repo ontology:

    opentofu-infra/
      AGENTS.md
      README.md
      docs/
        inventory.md
        state-and-backends.md
        migration-log.md
        decommission/
          taskrally.md
          openclaw-cloud.md
      stacks/
        active/
          jjpcodes/
            aws-static-sites/
          xuezh/
            azure-speech/
          gohome/
            hetzner-home-services/
        retired/
          taskrally/
            inventory/
          openclaw-cloud/
            inventory/
        candidates/
          README.md
      modules/
        aws-static-site/
        azure-speech/
        hcloud-host/
      scripts/
        inventory
        plan

Tree rationale:

- `stacks/active` answers "what cloud things are live?"
- `stacks/retired` answers "what used to exist and needs/has decommission evidence?"
- `stacks/candidates` is a holding area only if inventory finds stacks that are unclear; it should be emptied or deleted once classified.
- Stack names start with product/domain purpose, not provider. Provider appears in the leaf stack name, for example `aws-static-sites` or `azure-speech`.
- `modules` contains only reused OpenTofu modules. If a module is only used once and does not simplify the root, keep it inline.
- `docs/decommission` contains human-readable reports before any destructive action.
- `scripts` exists only for repeated safe operations, not one-off glue.

Stack-local shape:

    stacks/active/jjpcodes/aws-static-sites/
      AGENTS.md
      README.md
      backend.tf
      providers.tf
      main.tf
      variables.tf
      outputs.tf
      imports.md

Each stack README must say:

- What this stack owns.
- What it must not own.
- Backend/state location.
- Required provider credentials.
- Safe commands for `fmt`, `validate`, `plan`.
- A rollback note or restore path if state migration fails.

## Plan of Work

First, align on the repo name and ontology. Use `opentofu-infra` unless Josh explicitly chooses `cloud-infra`. Confirm the tree above. Confirm whether `eastwindlegal-nl` moves in the first website migration or later with a separate report.

Second, choose state strategy before moving any stack. The plan should inventory current state first and then propose one of:

- keep each existing backend during the first move, only moving files and docs;
- migrate to one central remote backend layout;
- keep local state only for tiny personal stacks, with encrypted backup and clear risk notes.

No state backend change should happen in the same step as a stack move unless the current state layout forces it. State backups come first.

Third, create the central repo skeleton. Add root `AGENTS.md`, `README.md`, docs placeholders, and empty stack directories only for approved categories. Do not pre-create a large taxonomy. The initial tree should be small enough to understand at a glance.

Fourth, inventory existing OpenTofu roots. For each root, collect:

- repo and path;
- provider(s);
- backend/state location;
- workspace, if any;
- resources from state, if state is available;
- DNS names/domains owned;
- obvious monthly-cost resources;
- whether the stack is active, retired, duplicate, or unknown;
- recommended action: migrate, keep for now, retire report, or ignore.

This inventory is read-only. Do not run `tofu apply` or `tofu destroy`.

Fifth, migrate `jjpcodes.com` infrastructure first. Copy the existing `website/infra/tofu/jjpcodes-blog` root into `stacks/active/jjpcodes/aws-static-sites` or a more precise leaf name if inventory shows multiple static-site stacks. Preserve state backend first. Run `tofu fmt`, `tofu validate`, and `tofu plan`. The acceptable first migration result is a no-op plan or a clearly explained provider/version-only diff that Josh approves. Then update `website/AGENTS.md` and website docs to say OpenTofu now lives in `opentofu-infra`, while website content and deploy scripts remain in `website`.

Sixth, handle `eastwindlegal-nl`. If it is an active static site controlled by Josh, migrate it next under `stacks/active/eastwindlegal/aws-static-site` or under `stacks/active/jjpcodes/` only if it is truly part of the same domain/product ownership. If it is stale, inventory and ask before moving.

Seventh, move xuezh Azure Speech OpenTofu. The xuezh app keeps code, config docs, and runtime env checks. The central repo owns the Azure Speech account. Update xuezh docs and AGENTS to point at the new owner. Do not touch the other agent's OpenClaw environment-variable work.

Eighth, write Task Rally decommission report. Inventory all Task Rally roots and any current resources. Produce a report with:

- resources still present by provider;
- state files/backends involved;
- dependencies or shared resources that should not be destroyed;
- estimated blast radius;
- proposed destroy order;
- exact commands that would be run later;
- "stop here for approval" line.

Ninth, write OpenClaw Cloud decommission report the same way. Treat it as separate from Task Rally because it may have separate states and earlier resources.

Tenth, update agent discovery docs. At minimum:

- central repo root `AGENTS.md`;
- `/Users/josh/AGENTS.md`, if this is the active global file;
- the deployed global preamble source in `~/code/nix/ai-stack` or wherever inspection proves it lives;
- affected repo AGENTS files in `website`, `xuezh`, `taskrally`, and `openclaw-cloud`.

The docs should be short and directive: cloud/provider resources live in `opentofu-infra`; app repos should not add new OpenTofu roots except temporary staging with explicit plan; Nix owns machine runtime; secrets stay in nix-secrets.

## Concrete Steps

Run from `/Users/josh/code` unless a command says otherwise.

Read the current tree and likely roots:

    find /Users/josh/code -path '*/.terraform' -prune -o -path '*/node_modules' -prune -o -name '*.tf' -print
    find /Users/josh/code -path '*/.terraform' -prune -o -name '*.tofu' -print

For each candidate root, collect a read-only inventory:

    tofu -chdir=<root> fmt -check -diff
    tofu -chdir=<root> init -backend=false
    tofu -chdir=<root> validate

Only run state commands where state is local or the backend is already configured and credentials are present:

    tofu -chdir=<root> state list

Create the central repo only after Josh approves the name:

    cd /Users/josh/code
    mkdir opentofu-infra
    cd opentofu-infra
    git init

Do not create a GitHub repo or push until the skeleton has been reviewed and committed locally.

For website migration, preserve state first. If state is local:

    cp terraform.tfstate terraform.tfstate.backup-<timestamp>

If state is remote, record backend config and run a plan from the old root before moving:

    tofu -chdir=/Users/josh/code/website/infra/tofu/jjpcodes-blog plan

After copying to the central repo, run the same plan from the new root. Do not delete the old root until the new root produces an acceptable plan and docs point to the new owner.

For retired stacks, write reports first under:

    docs/decommission/taskrally.md
    docs/decommission/openclaw-cloud.md

Do not run:

    tofu destroy

until Josh explicitly approves the specific report and stack.

## Validation and Acceptance

The plan is acceptable when:

1. Josh approves the repo name and top-level tree.
2. `tree -L 4` in the central repo makes ownership obvious without reading a long README.
3. Every moved active stack has a README explaining ownership, backend, credentials, and safe commands.
4. `jjpcodes.com` migration has a no-drift or approved-drift `tofu plan`.
5. `website` docs and AGENTS no longer imply website owns OpenTofu state.
6. xuezh Azure Speech OpenTofu is moved or has an approved reason to wait.
7. Task Rally and OpenClaw Cloud each have a written inventory and destroy report.
8. No destructive command has run before Josh approves the matching report.
9. Global and affected AGENTS docs point agents to the central OpenTofu owner.
10. There are no new cloud-resource OpenTofu roots left in app repos except explicitly documented temporary staging roots.

## Idempotence and Recovery

All first-pass work is read-only inventory and docs. Moving a stack must be reversible:

- copy files before deleting old roots;
- back up local state files before touching them;
- record remote backend settings before changing backend config;
- run old-root and new-root plans before deleting old-root files;
- keep old roots in place until the central root is validated;
- use Git commits at each working, reviewed slice.

Destroy work is deliberately separate. A destroy report is not approval. Approval must name the stack and report version.

## Artifacts and Notes

Known initial roots to inventory:

    /Users/josh/code/website/infra/tofu/jjpcodes-blog
    /Users/josh/code/website/infra/tofu/eastwindlegal-nl
    /Users/josh/code/xuezh/infra/azure/speech
    /Users/josh/code/taskrally/apps/cloud/infra/opentofu/envs/*
    /Users/josh/code/openclaw-cloud/infra/opentofu/envs/*
    /Users/josh/code/gohome/infra/tofu
    /Users/josh/code/djtbot/infra/opentofu/*
    /Users/josh/code/trmnl-inkhub/infra/tofu
    /Users/josh/code/nix/nixos-config/stacks/ai/infra/gcp/moltbot

Do not assume this list is complete. The inventory command in Concrete Steps is the source of truth for the first report.

## Interfaces and Dependencies

Central repo responsibilities:

- OpenTofu stack files.
- OpenTofu modules used by multiple stacks.
- State/backend documentation.
- Inventory and decommission reports.
- Provider-specific cloud resource ownership.

Application repo responsibilities:

- Application code.
- App deploy scripts.
- App runtime docs.
- References to central infra outputs.
- No new long-lived OpenTofu roots unless Josh approves an exception.

Nix repo responsibilities:

- Machine configuration.
- Runtime services.
- launchd/systemd.
- Caddy/Tailscale/service ingress on hosts.
- Secret consumption paths.

Secrets repo responsibilities:

- Secret values.
- Age/SOPS/secret material.
- No OpenTofu state unless explicitly designed and documented later.

Revision note: This ExecPlan was created from Josh's infra alignment request on 2026-05-06. It intentionally stops before any state move, cloud mutation, or retired-stack destroy.
