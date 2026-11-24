# function-approve end-to-end tests

This directory is a small Upbound project used only to run **real** end-to-end
tests for `function-approve`: `up test run tests/* --e2e` installs it on a
throwaway control plane, together with the function built from the pull request,
and plays a user and an operator through two approval rounds. It is never
published to production. No cloud account is needed.

## Layout

```
e2e/
├── upbound.yaml                 # test configuration; depends on the function under test (pinned by CI)
├── apis/xapprovals/             # XRD + Composition: a ConfigMap rendered by function-go-templating, then function-approve
├── functions/.gitkeep           # must exist for `up`
└── tests/
    └── e2etest-approve/
        ├── test.yaml            # E2ETest: one XR, waits for Ready + FunctionSuccess
        ├── approve.sh           # pre-assert-hook: gate holds, then approve
        └── change.sh            # post-assert-hook: change, gate holds again, approve, change applied
```

Dependency-installed functions are named `<org>-<repo>`, so the Composition
references `solutions-function-approve` and
`crossplane-contrib-function-go-templating`.

## What the test proves

The Composition renders a ConfigMap from `spec.resources.data`, then
function-approve guards `spec.resources`. The render step also writes
`status.renderedKeys` to the XR, so the test covers a pipeline where an earlier
step sets XR status, as patch-and-transform `ToCompositeFieldPath` patches commonly
do.

1. **Create:** the XR waits for approval (`ApprovalRequired=False`,
   `WaitingForApproval`) and nothing is composed (`approve.sh`).
2. **Approve** through the status subresource, as the README documents:
   `kubectl patch --subresource=status -p '{"status":{"approved":true}}'`. up test
   then waits for `Ready` and `FunctionSuccess`.
3. **Change** `spec.resources` (`change.sh`): a new approval request names the
   hash approved in step 2, and the ConfigMap keeps the approved data (`v1`).
4. **Approve** again: the ConfigMap gets the new data (`v2`).

## Hooks

`approve.sh` and `change.sh` are uptest hooks, set by the
`uptest.upbound.io/pre-assert-hook` and `post-assert-hook` annotations on the XR.
They run on the test runner with `kubectl` pointed at the test control plane.
Spaces doesn't allow Jobs or Pods on a control plane, so the approval can't run
inside it.

The hook paths are relative to the temp directory up test copies the manifest
to, which it creates directly under `$TMPDIR`; the scripts aren't copied. The e2e
workflow sets `TMPDIR` to `e2e/.tmp`, so `../../tests/e2etest-approve/approve.sh`
resolves to the script here. Locally:

```sh
cd e2e && mkdir -p .tmp && TMPDIR=$PWD/.tmp up test run tests/* --e2e
```

If up test changes that layout, the hook fails with `no such file or directory`;
it can't pass silently.

## Setup in the test (`initResources`)

A ClusterRole aggregated to Crossplane so it can compose ConfigMaps, which it
can't by default. Its verbs are explicit: the test user can only grant verbs it
holds itself, so `*` is rejected.

## Two repos, one function

The function dev build is pushed to `solutions/<repo>` and the wrapper project to
**`solutions/<repo>-e2e-wrapper`**. They must be separate: Crossplane's package
lock keys dependency-graph nodes by source repo, so a project and a function
sharing one repo collide (`node ... already exists`). The `<repo>-e2e-wrapper`
repo is granted to the e2e robot in `solutions-gitops-prod`.

## What runs in CI

On a pull request labelled `run-e2e-tests`, `.github/workflows/e2e.yaml` (synced
from `upbound/sa-up`):

1. builds the function package from the PR head and pushes it to `solutions/<repo>`
   at a dev tag,
2. pins that tag in `upbound.yaml` (`@FUNCTION_VERSION@`),
3. switches to the e2e Spaces control plane group, and
4. runs `up test run tests/* --e2e` with `TMPDIR=e2e/.tmp`, pushing the wrapper
   project to `solutions/<repo>-e2e-wrapper`.

A separate job then posts the `e2e` commit status that branch protection requires.
