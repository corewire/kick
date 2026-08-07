# Product and scope

## Purpose

KICK ensures that Kubernetes workloads receive current Secret and ConfigMap values when Kubernetes itself does not trigger a rollout.

A workload can appear healthy and its GitOps owner can appear reconciled while running Pods still contain older environment-variable values or application state loaded from mounted files. KICK detects this hidden restart requirement and performs the restart when policy allows — immediately by default, or gated on a GitOps provider or maintenance windows when configured.

## Product statement

> KICK triggers the restart Kubernetes does not know is required.

## Primary flow

1. A Secret or ConfigMap's relevant content changes.
2. KICK finds every in-scope Deployment that currently consumes the object.
3. KICK records or updates one pending restart request per Deployment.
4. KICK resolves the Deployment's GitOps owner.
5. The provider adapter evaluates reconciliation state and scheduling constraints.
6. When allowed, KICK re-reads the Deployment and all current dependencies.
7. KICK restarts only if at least one dependency is still newer than the current rollout.

## Initial scope

KICK v1 MUST support:

- namespaced `apps/v1` Deployments;
- automatic discovery of all supported Secret and ConfigMap references;
- change observation for Secret and ConfigMap content;
- Argo CD Application ownership discovery;
- AppProject sync-window evaluation;
- waiting for the Application to be Synced and idle;
- durable pending requests;
- normal Deployment rollout restart;
- controller restart recovery;
- metrics, events, conditions, Helm installation, CI, and e2e tests.

## Non-goals for v1

KICK v1 MUST NOT implement:

- StatefulSet or DaemonSet support;
- cross-namespace dependency references;
- arbitrary workload kinds;
- Secret rotation itself;
- direct Pod deletion or eviction;
- application-specific live reload protocols;
- image-pull Secret restarts;
- dependency hashes injected into Pods or workloads;
- Git write-back;
- a web UI;
- Flux support;
- manual approval unless separately accepted as an extension.

## Success criteria

The implementation is successful when an e2e test demonstrates:

1. a Deployment consumes a Secret;
2. the Secret changes;
3. the owning Argo CD Application is outside an allowed sync window;
4. KICK waits;
5. the window opens and Argo CD becomes Synced;
6. KICK re-evaluates state;
7. KICK creates exactly one new Deployment rollout if it is still required.
