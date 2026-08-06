# Argo CD sync-window evaluation

## Goal

Evaluate AppProject sync windows with behavior compatible with the supported Argo CD versions.

## Input

- resolved Application;
- resolved AppProject;
- current time;
- Application name;
- destination namespace;
- destination cluster name or server;
- AppProject `spec.syncWindows`.

## Required semantics

The adapter MUST account for:

- allow windows;
- deny windows;
- overlapping windows;
- deny precedence;
- application matching;
- destination namespace matching;
- destination cluster matching;
- wildcard behavior;
- time zones;
- applicable matching mode and any supported AND/OR semantics;
- next time at which the decision may change.

When no window applies, the schedule gate is allowed.

When an applicable deny window is active, the gate is denied.

When applicable allow windows exist, behavior outside those windows MUST match Argo CD semantics for the supported version.

## Compatibility strategy

Prefer one of:

1. reuse Argo CD API types and tested semantics;
2. isolate a compatible evaluator with fixtures copied from documented Argo CD cases;
3. use Argo CD's own library if its dependency footprint and stability are acceptable.

Do not implement an approximate cron check and call it compatible.

## Requeue behavior

`RequeueAt` is advisory. On every reconcile, evaluate current windows again.

AppProject changes MUST immediately re-enqueue pending requests associated with the project.

## Manual-sync and health semantics

Whether `manualSync` changes KICK's permission model is an explicit compatibility decision and remains open until verified.

KICK does not require Application health to be Healthy by default.

## Acceptance criteria

- Unit fixtures cover no windows, allow-only, deny-only, overlap, timezone, and selectors.
- Deny overrides allow where Argo CD does.
- AppProject edits alter pending decisions without waiting for an old timer.
- Tests are versioned against every supported Argo CD minor range.
