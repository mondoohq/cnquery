---
name: provider-verification
description: >-
  Verify mql provider resource/field changes against real cloud infrastructure.
  Given a pull request or a commit range, this provisions Terraform infra in the
  affected cloud(s), runs mql queries against every new or changed resource and
  field, reports the hourly cost (pausing for approval above $2/hr), opens a fix
  PR for any provider bugs it uncovers, and tears the infrastructure back down.
  Use this whenever someone wants to test, verify, smoke-test, or "prove out" a
  provider PR or a range of commits against live cloud APIs — e.g. "verify PR
  #7701 works", "spin up infra to test the new GCP resources", "check the azure
  changes against real infrastructure", "test resources changed between these
  commits". Trigger it even when the user only says "test this PR" in the
  context of an mql provider change.
---

# Provider Verification

mql resources are thin wrappers over cloud APIs, so a `.lr` schema change is
only really "done" once it has been run against a real cloud account. This
skill automates that loop: figure out what changed, stand up just enough real
infrastructure to exercise it, query it with `mql run`, and clean up.

It exists because the failure modes that matter — a wrong location wildcard, a
missing API view, an SDK that lags the service — are invisible to compile
checks and unit tests. They only show up against live APIs.

## Inputs

One of:
- **A pull request**: a number (`7701`) or URL. Diff comes from `gh pr diff`.
- **A commit range**: `<base>..<head>` (e.g. `7bfc8787a..HEAD`). Diff comes from
  `git diff`.

Accept multiple PRs at once — verify them together in one infra spin-up.

## Workflow

Work through these steps in order. Track them with TodoWrite — the run is long
and the teardown step must not be skipped.

### 1. Extract what changed

Run the bundled script — it parses the diff and lists every added/changed
resource and field, grouped by provider:

```bash
python3 .claude/skills/provider-verification/extract_changes.py --pr 7701
python3 .claude/skills/provider-verification/extract_changes.py --range A..B
```

It reports `.lr` schema changes (new resources, new fields) and flags the
provider `.go` files that changed. A PR can touch a provider's code without
touching its `.lr` (a pure bugfix); still verify those resources.

Doc-comment-only `.lr` changes (a PR that just adds doc-comments to existing
resources) do not need infrastructure — skip them, but say so in the report.

### 2. Build the affected providers

The merged PR code must be running locally. From the repo root:

```bash
make providers/build/<provider> && make providers/install/<provider>
```

Build every affected provider. If the PRs are **not** yet merged, check out the
PR branch first. If verifying a commit range, check out the head commit.

### 3. Check cloud auth

For each affected cloud, confirm credentials before provisioning anything:
`aws sts get-caller-identity`, `az account show`, `gcloud config get-value
project`, `oci iam region list`. If a cloud is not authenticated, stop and tell
the user — do not try to provision it.

### 4. Pre-flight: remove leftovers from prior runs

A run that crashed or was interrupted before teardown leaves tagged infra
behind — and it keeps billing until deleted. Clear it **before** provisioning
anything new. Everything this skill creates carries `project = mql-pr-verify`,
so leftovers are unambiguous. Two sources, cleanest first:

1. **Local scratch dirs with live state.** Any `~/dev/mql-verify-*/<cloud>/`
   whose `terraform state list` is non-empty is a prior run that never
   destroyed. Run `terraform destroy` there — intact state tears down in
   dependency order, which is more reliable than deleting resource-by-resource.

2. **Cloud tag sweep** (catches runs whose scratch dir or state was lost). For
   each authenticated cloud, list resources tagged `project = mql-pr-verify`,
   then delete them:

   ```bash
   # Azure — each run puts everything in one resource group
   az group list --query "[?tags.project=='mql-pr-verify'].name" -o tsv \
     | xargs -r -I{} az group delete -n {} --yes --no-wait

   # AWS — per region you provision in
   aws resourcegroupstaggingapi get-resources \
     --tag-filters Key=Project,Values=mql-pr-verify \
     --query 'ResourceTagMappingList[].ResourceARN' --output text

   # GCP — Terraform tags land as labels
   gcloud asset search-all-resources \
     --scope=projects/<project> --query='labels.project=mql-pr-verify' \
     --format='value(name,assetType)'
   ```

**List what matched before deleting.** If a match was created within the last
few minutes it may belong to a **concurrent** run — pause and confirm rather
than deleting it. AWS and GCP have no single "delete the whole tagged set"
command: delete by resource (see cloud-notes for per-service gotchas), or
`terraform destroy` via the leftover scratch dir when one still exists.

Report what was removed (or that nothing was) in the final report's Teardown
section.

### 5. Generate Terraform

Goal: the **cheapest** real resources that make each changed field return
non-empty, non-error data. Smallest SKUs, smallest instances, free tiers where
they exist.

Read `references/cloud-notes.md` before writing any Terraform — it lists the
per-cloud gotchas (SKUs that no longer exist, APIs that need a quota project,
resources Terraform cannot create) that this process has already hit. It will
save you a failed `apply`.

Write Terraform into a scratch directory **outside the repo** (e.g.
`~/dev/mql-verify-<timestamp>/<cloud>/`), one stack per cloud. When more than
one cloud is involved, dispatch one subagent per cloud to write its stack in
parallel — each agent writes the `.tf`, runs `terraform init/validate/plan`,
and reports a per-resource hourly cost. Agents must not `apply`.

Tag every resource with `project = mql-pr-verify` so leftovers are findable.

### 6. Cost gate

Sum the 1-hour cost across every cloud's `terraform plan`. Present a per-resource
cost table and the total.

- **Total ≤ $2/hour**: state the cost and proceed.
- **Total > $2/hour**: STOP. Show the table and ask the user to approve before
  applying. Do not apply until they say yes.

### 7. Apply

`terraform apply -auto-approve` per cloud — run them in parallel in the
background; some resources are slow (see cloud-notes). Re-apply on transient
failures, but cap it at **2–3 attempts** — a failure that survives that many
re-applies is not transient. Treat it as a blocker or an environment
limitation and stop retrying. If a resource genuinely cannot be created,
remove it from the stack, note it, and continue — one bad resource must not
block the rest.

### 8. Verify with `mql run`

For **every** new or changed resource and field, run a query and confirm two
things: it returns **no error**, and it returns **appropriate data**.

```bash
mql run <provider> -c "<resource> { <changed fields> }"
```

- A new field that resolves to `false`/`""`/`[]` is fine **if** that genuinely
  reflects the resource's state (feature disabled, list empty). It is **not**
  fine if the field should have data — that is a bug.
- A query that errors, or `no data available` caused by an underlying API
  error, is a bug. Capture the exact error.
- Typed reference fields (`vpc()`, `kinesisStream()`, …) must resolve to the
  referenced resource, not error.

For resources Terraform cannot provision (ephemeral jobs, etc.), create them
best-effort via the cloud CLI (see cloud-notes) so the accessor still gets
exercised. If even that is impossible, verify the accessor resolves cleanly
(empty, no error) and say so.

### 9. Triage bugs

A bug is any verification failure caused by **provider code**, including
preexisting bugs in code outside the PRs under test. For each:

- **Has a clear, verifiable fix**: fix it (see step 10).
- **No confident fix** (e.g. an SDK lagging a new API): do **not** guess a fix.
  Record it in the report and offer to open a tracking GitHub issue.

A failure caused by the cloud account (expired trial, missing quota, un-enabled
API) is **not** a provider bug — report it as an environment limitation.

### 10. Fix PR

If there are bugs with verifiable fixes, open **one combined PR** for all of
them, across every provider:

1. Work in a worktree branched from `main`.
2. Apply the fixes. Match existing patterns in the provider (see CLAUDE.md).
3. `gofmt -w` changed files; rebuild the provider; **re-run the failing
   queries against the still-live infrastructure** to confirm each fix works.
4. Commit `*.permissions.json` if a fix changed it. No `.lr.versions` change
   unless a fix adds a schema field.
5. Commit (emoji-prefixed per CLAUDE.md — `🐛`), push, `gh pr create`.

Verify fixes **before** teardown — the infrastructure is needed to prove them.

### 11. Teardown — always

Always destroy every stack at the end, even on failure, even when bugs were
found (the fix PR was already verified in step 10). Run `terraform destroy` per
cloud.

Destroy is fragile — handle the known failure modes in cloud-notes (orphaned
EFS mount targets blocking AWS subnets, OCI API circuit-breakers, resources
Terraform dropped from state). Fall back to CLI deletion when `terraform
destroy` cannot finish. Confirm nothing tagged `mql-pr-verify` remains.

Note anything that genuinely cannot be deleted (e.g. an App Engine app) in the
report — do not leave the user guessing.

### 12. Report

Always end with this structure:

```
# Verification report — <PRs / commit range>

## Provisioned
<per-cloud resource count + total $/hour for the run>

## Results
| PR / area | Resource / field | Result | Detail |
(✅ pass / ⚠️ partial / ❌ bug, one row per changed resource or field group)

## Bugs
<each bug: file:line, the failing query, observed vs expected, and either
"fixed in PR #NNNN" or "no verified fix — offer to open an issue">

## Environment limitations
<failures caused by the account, not the code>

## Teardown
<confirmation everything was destroyed; anything that could not be>
```

After the report, if there are unfixable bugs, offer to open a tracking issue.

## Key rules

- **Cost gate is $2/hour total.** Below it, proceed. Above it, pause for
  approval. Always show the number.
- **One combined fix PR** for all bugs, regardless of how many providers.
- **Always tear down** — unconditionally, at the end.
- **Cheapest infra that works.** This is a verification run, not a deployment.
- **Honest reporting.** An empty result is a pass only when empty is correct;
  otherwise it is a bug. Never claim a field works without seeing real data.

## Reference

- `references/cloud-notes.md` — per-cloud Terraform/CLI gotchas, slow resources,
  and what cannot be provisioned or destroyed. Read it before steps 4, 5, 7, 11.
- `extract_changes.py` — diff → changed resources/fields, grouped by provider.
