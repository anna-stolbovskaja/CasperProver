# SDK release rehearsal

> **Status.** Living checklist. This document is the *operator's manual*
> for cofounders and maintainers who need to verify that the SDK release
> pipeline still works, *before* cutting the next real release. It is
> intentionally hand-runnable through the GitHub UI so a non-Go / non-npm
> operator can do it end-to-end.
>
> **Scope.** This document does not re-explain how the pipeline works —
> the source of truth for that is [`sdk/PUBLISHING.md`](../../sdk/PUBLISHING.md).
> Read that first. This is the *rehearsal* and *gap-audit* on top of it.

## TL;DR

Every 60 days (or before any release after ≥ 30 days of quiet), do:

1. **Manifest sync check** — [Section 2](#2-manifest-sync-check).
2. **Per-workflow dispatch dry-run** — [Section 3](#3-per-workflow-dispatch-dry-run).
3. **End-to-end release-please walkthrough** — [Section 4](#4-end-to-end-release-please-walkthrough).
4. **Environment / trusted-publisher config audit** — [Section 5](#5-environment--trusted-publisher-config-audit).
5. **Local-clone tag hygiene** — [Section 6](#6-local-clone-tag-hygiene).

Total time: ~30 minutes if nothing has drifted, ~2 hours if setting up
from scratch after a fresh clone.

---

## 1. What "release" means, in one paragraph

Merging a `chore(sdk-*): release X.Y.Z` PR (opened by release-please)
causes release-please to cut a GitHub Release with a tag matching
`sdk-ts-vX.Y.Z` / `sdk-py-vX.Y.Z` / `sdk/vX.Y.Z`. That Release event
fires `sdk-publish-npm.yml`, `sdk-publish-pypi.yml`, or
`sdk-publish-go.yml` respectively. Each publish workflow re-runs the
`build-and-verify` job on the tagged commit before invoking its
registry step. Environment approval (`npm-publish` / `pypi-publish`)
gates the final publish.

There is no auto-publish on `main` and no long-lived PyPI token — see
`sdk/PUBLISHING.md` for the design rationale.

---

## 2. Manifest sync check

The pipeline has **four** places where an SDK version can live. They
must agree at rest.

| Location                                | Owner (writes)             | Consumer (reads)                |
|-----------------------------------------|----------------------------|---------------------------------|
| `sdk/typescript/package.json` → `version` | release-please             | npm publish, sdk-semver-check   |
| `sdk/python/pyproject.toml` → `version`   | release-please             | PyPI publish, sdk-semver-check  |
| `sdk/version.txt`                         | *nobody* (see gap below)   | sdk-semver-check                |
| `.release-please-manifest.json`           | release-please             | release-please only             |

### Rehearsal step 2a — the four-way diff

Run this from the repo root; every command below is copy-pasteable.

```sh
ts=$(node -p "require('./sdk/typescript/package.json').version")
py=$(python3 -c "import tomllib; print(tomllib.load(open('sdk/python/pyproject.toml','rb'))['project']['version'])")
vtxt=$(cat sdk/version.txt)
mts=$(jq -r '."sdk/typescript"' .release-please-manifest.json)
mpy=$(jq -r '."sdk/python"'     .release-please-manifest.json)
mgo=$(jq -r '."sdk"'            .release-please-manifest.json)
printf "ts pkg=%s  manifest=%s\npy pyproject=%s  manifest=%s\ngo version.txt=%s  manifest=%s  latest-tag=%s\n" \
       "$ts" "$mts" "$py" "$mpy" "$vtxt" "$mgo" \
       "$(git tag -l 'sdk/v*' | sort -V | tail -n1)"
```

**Pass criterion.** For each SDK, the two visible sources of truth
(manifest + package/pyproject) must be equal, and the newest `sdk/v*`
tag on `origin/main` must be ≤ the manifest.

### Gap G-1 — `sdk/version.txt` is not release-please's file

The release-please config is set to `release-type: "go"` for the `sdk/`
component and does not have `sdk/version.txt` in `extra-files`. That
means release-please **does not update `sdk/version.txt`** when it bumps
the Go SDK — the file drifts every release. Currently:

```text
sdk/version.txt         → 0.1.0    (drifts)
release-please manifest → 0.1.2    (source of truth for Go)
```

`sdk-semver-check.yml`'s `extract_go` reads `sdk/version.txt`, so on
every Go release PR its "old" and "new" are both the stale `0.1.0` and
the diff is a no-op — semver-check silently passes without ever
enforcing the Go bump rules. **This is a real hole; sdk-py and sdk-ts
are checked, sdk-go isn't.**

**Action.** In a future PR, either:

- (a) add `sdk/version.txt` to the `extra-files` of the `sdk` component
  in `.release-please-config.json` so release-please starts writing it,
  or
- (b) rewrite `extract_go` in `sdk-semver-check.yml` to read the
  `.release-please-manifest.json`'s `"sdk"` field instead of
  `version.txt`.

Option (b) is preferred — single source of truth. Do not do this inside
the rehearsal PR — it is a semver-guard behaviour change that needs its
own PR + human review.

---

## 3. Per-workflow dispatch dry-run

Every publish workflow has a `workflow_dispatch` trigger with
`dry_run: true` as the default. Doing all three from the GitHub UI is
the fastest way to catch a broken package.json / stale test / missing
build step **without** touching a real registry.

### 3a. npm (`sdk-publish-npm.yml`)

1. GitHub UI → **Actions** → **sdk-publish-npm**.
2. Click **Run workflow** on the `main` branch, leave `dry_run=true`,
   run.
3. Expected: `build-and-verify` job goes green; `publish` job is
   **skipped** (its `if:` requires either a release event or
   `dry_run=false`).
4. Open the run logs, expand *Pack (dry-run)*. Confirm:
   - `npm notice filename:  @casperprover-sdk-<version>.tgz`
   - `dist/index.js` and `dist/index.d.ts` show up in the "Tarball
     Contents" list.

**Failure modes seen historically.** Node version too old (needed
≥ 22.6 for `--experimental-strip-types`), missing
`repository`/`homepage`/`bugs` in `package.json` (npm provenance
requires these — see commit `cee0b17`), `dist/` not in the tarball
because `tsconfig.json` output path drifted.

### 3b. PyPI (`sdk-publish-pypi.yml`)

1. Actions → **sdk-publish-pypi** → Run workflow.
2. Leave `dry_run=true`, leave `test_pypi=false`.
3. Expected: `build-and-verify` goes green, `publish` skipped.
4. In logs, expand *Sanity — wheel actually contains the package*.
   Confirm:
   - `casperprover/client.py` present
   - `casperprover/receipt.py` present
   - `casperprover/tests` **absent** (the guard `grep -q 'casperprover/tests' && exit 1`)
5. Expand *twine check*: every file must report `PASSED`.

**Failure modes seen historically.** Test files leaking into the wheel
(`pyproject.toml` `[tool.hatch.build]` `exclude` drifted), README
missing (twine emits a warning that turns into a fail), Python version
too old.

### 3c. Go (`sdk-publish-go.yml`)

1. Actions → **sdk-publish-go** → Run workflow.
2. Leave `dry_run=true`.
3. Expected: `build-and-verify` green, `proxy-warmup` skipped.
4. In logs, expand *Verify module*. Confirm:
   - `go mod tidy -diff` produces no diff
   - `go vet ./...` clean
   - `go test -race ./...` all pass

**Failure modes seen historically.** Module path in `sdk/go.mod`
diverges from the canonical
`github.com/anna-stolbovskaja/CasperProver/sdk` (see `ff51334`); toolchain
directive in `go.mod` incompatible with the runner's Go.

### 3d. Semver-check (`sdk-semver-check.yml`)

Not directly dispatchable — it runs on PR events. To rehearse it, open
a throwaway branch that bumps `sdk/typescript/package.json` from
`0.1.2` → `1.0.0` **without** `!` or `BREAKING CHANGE:` in the PR
title / body, and open a *draft* PR. The `sdk-semver-check` check must
fail with `sdk/typescript MAJOR bump 0.1.2 -> 1.0.0 requires a '!'
commit or 'BREAKING CHANGE:' footer`. Close the draft without merging.

**Failure mode.** Regex in the PR-title matcher accepts `Foo(bar):
baz` — verify by opening a draft PR with a title like
`nope: whatever` and confirming that the check fails on the title.

---

## 4. End-to-end release-please walkthrough

This is the *real* rehearsal — not a workflow dry-run but the whole
chain, using a no-op commit. Doing it once every 60 days catches drift
between release-please, the manifest, and the publish workflows.

### 4a. Create a rehearsal commit

On a rehearsal branch off `main` (name it e.g.
`rehearsal/release-please-YYYY-MM-DD`), land a *docs-only* commit that
release-please considers releasable **for exactly one SDK**. Simplest:

```sh
# Trigger release-please to open a sdk-ts release PR
git checkout -b rehearsal/release-please-$(date +%Y-%m-%d)
printf '\n<!-- release-please rehearsal $(date +%Y-%m-%d) -->\n' >> sdk/typescript/README.md
git add sdk/typescript/README.md
git commit -m "docs(sdk-ts): rehearsal comment ($(date +%Y-%m-%d))"
git push -u origin HEAD
gh pr create --title "docs(sdk-ts): rehearsal comment" --body "Docs-only. Triggers a release-please PR after merge — see docs/roadmap/SDK_RELEASE_REHEARSAL.md §4."
```

Merge the PR normally. Do **not** cherry-pick or squash it in a way
that strips the Conventional Commit type — the whole point is that
release-please sees `docs(sdk-ts):` in the merged history.

### 4b. Expected outcome

Within a minute of the merge landing on `main`:

- `sdk-release-please` workflow fires.
- release-please opens a new PR titled
  `chore(sdk-ts): release 0.1.3` (or the next patch version), and it
  updates:
  - `sdk/typescript/package.json` `version` → `0.1.3`
  - `sdk/typescript/CHANGELOG.md` with the `docs(sdk-ts):` bullet
  - `.release-please-manifest.json` `"sdk/typescript"` → `0.1.3`

If instead release-please opens sdk-py or sdk-go PRs, or opens no PR,
that is a real drift — stop and investigate before touching a real
release.

### 4c. Approval gate

The release-please PR is the human decision point. Options:

- **Rehearsal only.** Close the PR without merging; leave a comment
  linking to this doc. The `chore(sdk-ts): release 0.1.3` version
  bump is *not* committed.
- **Real release.** Merge it. release-please then cuts
  `sdk-ts-v0.1.3` as a GitHub Release, which fires
  `sdk-publish-npm.yml` in release mode. Approve the `npm-publish`
  environment prompt to actually publish.

**Rehearsal choice.** During a scheduled rehearsal, close without
merging. Only merge if the rehearsal was scheduled *because* we
wanted to release anyway.

---

## 5. Environment / trusted-publisher config audit

Two GitHub Environments must exist for `publish` jobs to succeed.

### 5a. `npm-publish` environment

- GitHub UI → **Settings → Environments → New environment** → name
  `npm-publish`.
- Required reviewers: at least one maintainer (Anna, Bomani).
- Secret: `NPM_TOKEN` — an *automation* token owned by an
  `@casperprover` npm-org maintainer, scoped to publish
  `@casperprover/sdk`.
- Rotation: every 6 months, or immediately on maintainer change.

Verify by clicking **Environments → npm-publish** in Settings: the
required-reviewers list must be non-empty and `NPM_TOKEN` must show as
"Set on 20YY-MM-DD".

### 5b. `pypi-publish` environment

- GitHub UI → **Settings → Environments → New environment** → name
  `pypi-publish`.
- Required reviewers: at least one maintainer.
- **No secret required for CI publish.** `sdk-publish-pypi.yml` uploads
  via OIDC (`id-token: write` + `pypa/gh-action-pypi-publish` with no
  `password:` field). A long-lived PyPI API token in the vault
  (`team.pypi.casperprover_api`) may exist as a fallback for manual
  `twine upload` from a maintainer's laptop, but the CI pipeline does
  **not** read it.
- On https://pypi.org/manage/account/publishing/, add a *pending
  publisher* pointing at:
  - Repository: `anna-stolbovskaja/CasperProver`
  - Workflow filename: `sdk-publish-pypi.yml` (bare filename, not path)
  - Environment: `pypi-publish`
  - PyPI Project name: `casperprover`
- After the first successful upload, the pending publisher is promoted
  to an active trusted publisher automatically.

**Verify.** From the pypi.org UI, log in as a maintainer, go to
*Your projects → casperprover → Manage → Publishing*. There must be
exactly one trusted publisher entry matching the above four fields.
If the config exists but with a different `workflow` name, PyPI will
issue `HTTP 403 invalid-publisher` at publish time.

### 5c. Go — no environment needed

`sdk-publish-go.yml`'s `proxy-warmup` job runs unattended: no secret,
no environment, no approval. A tag on origin is the release; the job
just warms `proxy.golang.org`. If you delete a `sdk/v*` tag, the proxy
still serves the old artifact indefinitely — see the Go module
[deprecation guide](https://go.dev/ref/mod#module-cache) before ever
un-tagging.

---

## 6. Local-clone tag hygiene

Prior to `ff51334` (which fixed the Go tag format), release-please
emitted double-`v` tags: `sdk-go-vv0.1.2`, `sdk-py-vv0.1.2`,
`sdk-ts-vv0.1.2`. **These are no longer on origin** — verified
2026-07-31 via `git ls-remote --tags origin`; the only SDK tags on
remote are `sdk-py-v0.1.2` and `sdk-ts-v0.1.2`.

However, they *may still exist in an operator's local clone* if the
clone was fetched before the cleanup and `git fetch --prune-tags` has
never run against it. In that case:

```sh
git fetch --prune --prune-tags origin
git tag -l | grep sdk-      # confirm no `vv` remains
```

`--prune-tags` removes any local tag that no longer exists on origin.
Safe: no remote mutation, only local cleanup.

Add `git fetch --prune-tags` to your rehearsal preamble — otherwise the
§2 manifest check may falsely surface "garbage tags" that are only a
local-clone artifact.

---

## 7. First-real-release checklist

The very first non-rehearsal release goes through more scrutiny than
subsequent ones. Do this once, per registry:

### 7a. npm — first publish

- Confirm `@casperprover/sdk` is not already taken on npm.
- Confirm the `@casperprover` org exists and has at least two owners
  (bus-factor).
- Set the initial `dist-tag` to `latest` unless the release is a
  prerelease.
- After first successful publish, verify on
  https://www.npmjs.com/package/@casperprover/sdk that:
  - Provenance badge is present (`Verified` icon).
  - README renders.
  - `homepage` / `bugs` / `repository` links resolve.
- Install into a scratch project (`npm i @casperprover/sdk`) and
  import once to smoke-test.

### 7b. PyPI — first publish

- Confirm `casperprover` is not already taken on PyPI.
- The trusted-publisher config from §5b must be **pending publisher**
  (not "active") — PyPI promotes it on first successful upload.
- After publish, verify at https://pypi.org/project/casperprover/
  that the version, README, license, and long description render.
- `pip install casperprover==<version>` into a scratch venv and
  `import casperprover` to smoke-test.

### 7c. Go — first publish

- The tag is the release; nothing to configure on
  `proxy.golang.org`.
- Verify `go get github.com/anna-stolbovskaja/CasperProver/sdk@vX.Y.Z`
  resolves from a clean `GOMODCACHE`.
- Verify `pkg.go.dev/github.com/anna-stolbovskaja/CasperProver/sdk`
  shows the version (may take up to 30 min to index).

---

## 8. Known gaps (register)

Living register — every gap surfaced by a rehearsal lands here.
Resolved gaps stay in the table with a `resolved:` line so they don't
get re-discovered.

| ID  | Gap                                                                       | Impact                                                                 | Suggested fix                                                            | Status  |
|-----|---------------------------------------------------------------------------|------------------------------------------------------------------------|--------------------------------------------------------------------------|---------|
| G-1 | `sdk/version.txt` (`0.1.0`) is stale vs manifest (`0.1.2`); not in release-please `extra-files` | `sdk-semver-check` silently no-ops on Go SDK version bumps — a `feat: rewrite` could ship MAJOR without `!` / `BREAKING CHANGE:` | Rewrite `extract_go()` in `sdk-semver-check.yml` to read `.release-please-manifest.json["sdk"]` instead of `sdk/version.txt`; either delete `sdk/version.txt` or add it to release-please `extra-files` for `sdk` | open    |
| G-2 | Double-`v` `sdk-*-vv0.1.2` tags on origin                                 | Cosmetic if present                                                    | **Resolved 2026-07-31.** Verified via `git ls-remote --tags origin`: no `vv` tags on remote. Operators still see them locally must run `git fetch --prune-tags` — see §6 | resolved |
| G-3 | `npm-publish` / `pypi-publish` environments must be created out-of-band   | First release fails until an operator creates them in Settings         | Documented in §5; no in-repo automation possible                         | open    |
| G-4 | No end-to-end integration test — only per-workflow dry-run                | A drift between release-please and publish workflows can only be caught by manual rehearsal | The §4 walkthrough IS the mitigation; keep running it every 60 days     | open    |
| G-5 | PyPI trusted-publisher OIDC binding must be manually configured on pypi.org | First PyPI publish will fail with `HTTP 403 invalid-publisher` until §5b OIDC config exists; PyPI API tokens in vault do NOT paper over this because CI doesn't read them | Documented in §5b — one *pending publisher* row on pypi.org matching `repo=CasperProver`, `owner=anna-stolbovskaja`, `workflow=sdk-publish-pypi.yml`, `environment=pypi-publish` | open    |

Add a row every time a rehearsal or a real release surfaces something.

---

## Related documents

- [`sdk/PUBLISHING.md`](../../sdk/PUBLISHING.md) — how the pipeline is
  designed and why.
- [`.release-please-config.json`](../../.release-please-config.json) —
  release-please's configuration for the three SDKs.
- [`.github/workflows/sdk-publish-*.yml`](../../.github/workflows) — the
  three publish workflows.
- [`.github/workflows/sdk-semver-check.yml`](../../.github/workflows/sdk-semver-check.yml)
  — semver / Conventional Commits enforcement on every PR touching
  `sdk/**`.

---

## Reverification log

- **2026-07-31.** Re-checked all five gaps against live repo state:
  - G-1 confirmed real: `sdk/version.txt = 0.1.0`, manifest = `0.1.2`,
    release-please `extra-files: []` for `sdk`, `sdk-semver-check.yml`
    `extract_go()` reads `sdk/version.txt`. Semver-guard silently
    no-ops on Go.
  - G-2 downgraded to `resolved`: `git ls-remote --tags origin`
    returns only `sdk-py-v0.1.2` and `sdk-ts-v0.1.2`; no `vv` tags
    on origin. What looked like origin garbage was a local-clone
    artifact fixable by `git fetch --prune-tags`.
  - G-3 unchanged: requires manual verification in GitHub Settings
    UI (no CLI-inspectable via installer PATs).
  - G-4 unchanged: no in-repo automation possible; §4 walkthrough is
    the mitigation.
  - G-5 refined: workflow uses OIDC (`id-token: write`, no
    `password:` field), not the PyPI API tokens present in
    `team.pypi.casperprover_api`. Vault tokens are a manual-fallback
    path, not what CI reads. Gap is really the pypi.org
    pending-publisher config binding, not token provisioning.

---

## Change log for this document

| Date       | Change                                            | Author |
|------------|---------------------------------------------------|--------|
| 2026-07-31 | Initial rehearsal doc, gaps G-1 through G-5.      | Pancake / cofounder pipeline |
