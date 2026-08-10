---
title: Documentation
weight: 1
description: KICK operator documentation.
llmsDescription: |
  Documentation index for the KICK Kubernetes operator. Sections: getting
  started (install + quickstart), concepts (discovery, freshness, GitOps
  gates), theory (formal operator model with notation), guides (ArgoCD,
  external-secrets, troubleshooting), reference (KickPolicy/KickRequest,
  metrics, events, configuration), operations (RBAC, security, scalability,
  upgrades), development, and design decisions.
---

KICK restarts a workload when a `Secret` or `ConfigMap` it consumes changes —
but only when the running rollout is actually stale, and only when your GitOps
tool permits the restart.

### What is KICK?

Kubernetes never restarts a Pod when a `Secret` or `ConfigMap` it reads changes,
so the running Pod silently drifts from its intended configuration. KICK is a
small operator that closes this gap: it discovers each workload's dependencies,
detects relevant changes, checks that the running rollout predates the change,
optionally asks your GitOps tool for permission, and then issues exactly one
restart via the standard `kubectl.kubernetes.io/restartedAt` annotation. It
injects no state into your workloads.

### Where to start

- **[Install KICK](installation/)** with Helm (or from source for local dev).
- Run the **[Quickstart](quickstart/)** to see a Secret change restart a Deployment.
- Read **[Concepts](concepts/)** to understand discovery, freshness, and GitOps gating.
- Look up fields in the **[Reference](reference/)**.

## Sections

| Section | What you'll find |
|---------|------------------|
| [Installation](installation/) | Install KICK with Helm or from source |
| [Quickstart](quickstart/) | Watch KICK restart a Deployment on Kind |
| [Concepts](concepts/) | Discovery, freshness, and GitOps gating, explained |
| [Comparison](comparison/) | How KICK differs from Reloader and Wave |
| [Guides](guides/) | Argo CD, External Secrets, and troubleshooting |
| [Reference](reference/) | KickPolicy / KickRequest API, metrics, events, config |
| [Operations](operations/) | RBAC, security, scalability, upgrades |
| [Theory](theory/operator-model/) | The formal operator model in scientific notation |
| [Development](development/) | Debugging, the timeline UI, workflow |
| [For AI Agents](for-ai-agents/) | llms.txt, Markdown output, agent instructions |
| [Decisions](decisions/) | Architecture decision records |
