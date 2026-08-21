# 
# Argo CD Guide

KICK uses Argo CD as the first GitOps provider adapter.

## Ownership

Preferred signal: `argocd.argoproj.io/tracking-id` on workloads.

If tracking annotation is missing or invalid, fallback owner resolution is used.

Automatic restart is blocked when ownership is missing or ambiguous.

## Gates

KICK evaluates:

- Application sync state;
- AppProject sync windows.

Blocked gates keep the request in waiting phases until re-evaluation succeeds or policy changes.
