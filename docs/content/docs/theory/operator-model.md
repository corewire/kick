---
title: The KICK operator, formally
linkTitle: Operator model
weight: 10
math: true
description: A self-contained formal model of the KICK operator — state, observation, freshness, gating, the reconcile transition system, and its safety and liveness properties.
llmsDescription: |
  Formal model of the KICK operator in scientific notation. Defines the cluster
  state, dependency extraction, source fingerprints and the observation store,
  the freshness/staleness predicate, the gate function (native windows + GitOps
  provider), the KickRequest transition system, coalescing, and the restart
  action. States and proves the operator invariants: no spurious baseline
  restart, eventual restart on relevant change, kind-agnostic freshness,
  at-most-one active request per target, gate safety, and non-injection of
  workload state.
---

This page gives a precise, self-contained model of what the KICK controller
does. It is written for readers who want to reason about correctness rather than
read Go. Every definition corresponds to a concrete piece of the controller, and
the invariants at the end are the properties the implementation is designed to
preserve.

{{< callout type="info" >}}
Notation is standard set theory and first-order logic. We write \(2^{X}\) for
the power set of \(X\), \(f : X \rightharpoonup Y\) for a partial function, and
\(\lnot,\ \land,\ \lor,\ \Rightarrow\) for the usual connectives.
{{< /callout >}}

## 1. Objects and state

Let a point in time be \(t \in \mathbb{R}_{\ge 0}\). The observable cluster state
at time \(t\) is a tuple

$$
\mathcal{C}(t) = \big(\, W,\; S,\; \mathsf{data},\; \mathsf{tmpl},\; \mathsf{rs} \,\big),
$$

whose components are:

- \(W\) — the set of **workloads**. Each \(w \in W\) has a kind
  \(\kappa(w) \in \{\textsf{Deployment},\ \textsf{StatefulSet},\ \textsf{DaemonSet}\}\)
  and a namespace \(\mathsf{ns}(w)\).
- \(S\) — the set of **sources**, i.e. objects of kind
  \(\textsf{Secret}\) or \(\textsf{ConfigMap}\). Each \(s \in S\) has a creation
  time \(\gamma(s)\) and a resource version \(\rho(s) \in \mathbb{N}\).
- \(\mathsf{data}(s)\) — the key/value payload of a source (`data` +
  `binaryData`/`stringData`), together with its `type` and immutability flag.
- \(\mathsf{tmpl}(w)\) — the Pod template of a workload, in particular its
  annotation map \(\mathsf{tmpl}(w).\mathsf{ann}\).
- \(\mathsf{rs}(w)\) — for a Deployment, the set of its ReplicaSets; the *current*
  one is \(\mathsf{rs}^{\star}(w)\).

The controller never mutates \(\mathsf{data}\); its only write to cluster state
is a single annotation on \(\mathsf{tmpl}(w)\), defined in §7.

## 2. Dependency extraction

A workload consumes a source when its Pod template references it through an
environment variable, an `envFrom`, a volume, or a projected volume source, in
any container or init container. Image-pull secrets are **not** consumed data and
are excluded. This is captured by a pure function

$$
\mathsf{deps} : W \longrightarrow 2^{S},
\qquad
\mathsf{deps}(w) = \big\{\, s \in S \ \mid\ s \text{ is referenced by } \mathsf{tmpl}(w) \text{ as data} \,\big\}.
$$

\(\mathsf{deps}\) is deterministic and depends only on \(\mathsf{tmpl}(w)\); it is
insensitive to reference multiplicity, so a source referenced twice contributes
once. Its inverse image gives the **consumers** of a source,

$$
\mathsf{cons}(s) = \{\, w \in W \ \mid\ s \in \mathsf{deps}(w) \,\},
$$

restricted to \(\mathsf{ns}(w) = \mathsf{ns}(s)\).

## 3. Fingerprints and relevant change

To distinguish a change that matters from one that does not, each source is
reduced to a content **fingerprint** by a collision-resistant hash
\(H = \mathrm{SHA\text{-}256}\):

$$
\phi(s) \;=\; H\!\Big(\,\textsf{type}(s)\ \Vert\ \textsf{imm}(s)\ \Vert \mathop{\Big\Vert}\limits_{k \in \mathrm{sort}(\mathrm{keys})} \big(k \Vert \mathsf{data}(s)[k]\big)\Big) \ \in\ \{0,1\}^{256}.
$$

Keys are sorted so the fingerprint is canonical, and only the payload, type, and
immutability enter it. Object metadata — labels, annotations, `resourceVersion`,
managed fields — is deliberately excluded. Hence

$$
\phi(s) = \phi(s') \iff \mathsf{data}(s) = \mathsf{data}(s') \ \land\ \textsf{type}(s)=\textsf{type}(s') \ \land\ \textsf{imm}(s)=\textsf{imm}(s').
$$

A **relevant change** to a source is exactly a change of \(\phi\); a change of
\(\rho\) with fixed \(\phi\) is *metadata-only*.

## 4. The observation store

The controller maintains a durable partial map from a source identity to a
record,

$$
\Omega : \mathrm{Id}(S) \rightharpoonup \mathcal{R},
\qquad
\mathcal{R} = \big(\,\rho_{\text{seen}},\ \rho_{\text{rel}},\ \theta,\ \varphi\,\big),
$$

where \(\theta\) is the time of the last relevant change and \(\varphi\) the last
relevant fingerprint. When the controller observes a source \(s\) at wall-clock
time \(t\), the store transition \(\mathsf{obs}\) classifies the event and
updates the record:

$$
\mathsf{obs}(\Omega, s, t) =
\begin{cases}
\textsf{Baseline} & s \notin \operatorname{dom}\Omega, \\[2pt]
\textsf{NoChange} & \varphi = \phi(s)\ \land\ \rho_{\text{seen}} = \rho(s), \\[2pt]
\textsf{MetaOnly} & \varphi = \phi(s)\ \land\ \rho_{\text{seen}} \neq \rho(s), \\[2pt]
\textsf{Relevant} & \varphi \neq \phi(s).
\end{cases}
$$

The recorded change time is the crux of baseline correctness:

$$
\theta' =
\begin{cases}
\gamma(s) & \text{on }\textsf{Baseline}\quad(\text{the source's own creation time}),\\
t & \text{on }\textsf{Relevant}\quad(\text{observed “now”}),\\
\theta & \text{on }\textsf{NoChange},\ \textsf{MetaOnly}.
\end{cases}
$$

{{< callout type="info" >}}
**Why baseline uses \(\gamma(s)\) and not \(t\).** Anchoring the first
observation to the source's creation time makes a dependency that was created
*together with* its workload provably not-newer than that workload's rollout
(§5). Using the wall-clock observation instant instead would make freshness
depend on a sub-second race between "KICK first saw the Secret" and "the
ReplicaSet was created", producing spurious restarts at creation.
{{< /callout >}}

Only \(\textsf{Baseline}\) and \(\textsf{Relevant}\) enqueue work for the
consumers \(\mathsf{cons}(s)\); \(\textsf{NoChange}\) and \(\textsf{MetaOnly}\)
are inert.

## 5. Rollout state and the freshness relation

For a workload \(w\) the rollout inspector returns a **start time** and a
**completeness** flag,

$$
\mathsf{R}(w) = \big(\sigma(w),\ \mathrm{complete}(w)\big).
$$

The start time is the moment the currently-running Pod template came into being:

$$
\sigma(w) =
\begin{cases}
\mathsf{tmpl}(w).\mathsf{ann}[\textsf{restartedAt}] & \text{if that annotation is set},\\
\gamma\!\big(\mathsf{rs}^{\star}(w)\big) & \kappa(w)=\textsf{Deployment}\ \text{(current ReplicaSet)},\\
\gamma(w) & \kappa(w)\in\{\textsf{StatefulSet},\textsf{DaemonSet}\}.
\end{cases}
$$

Completeness is **kind-aware** — this is what lets non-Deployment workloads be
evaluated at all:

$$
\mathrm{complete}(w) =
\begin{cases}
\big(\mathsf{rs}^{\star}(w)\neq\varnothing\big)\ \land\ \lnot\text{InProgress}(w)\ \land\ \lnot\text{Failed}(w) & \kappa(w)=\textsf{Deployment},\\[4pt]
\mathrm{observedGen}(w) = \mathrm{gen}(w)\ \land\ \text{replicas up to date} & \kappa(w)\in\{\textsf{StatefulSet},\textsf{DaemonSet}\}.
\end{cases}
$$

Given the dependency scope \(D(w) \subseteq \mathsf{deps}(w)\) selected by the
policy (§6), the **latest relevant change** seen for \(w\) is

$$
\Lambda(w) \;=\; \max_{\,d \,\in\, D(w)\,\cap\,\operatorname{dom}\Omega} \Omega(d).\theta,
\qquad
\Lambda(w) = -\infty \ \text{ if the set is empty.}
$$

The workload is **stale** exactly when its rollout is complete yet older than the
latest relevant dependency change:

$$
\boxed{\ \mathrm{stale}(w) \;\iff\; \mathrm{complete}(w)\ \land\ \Lambda(w) > \sigma(w)\ }
$$

If \(\lnot\,\mathrm{complete}(w)\) the workload is *in progress* and no freshness
decision is taken (the request waits). The comparison \( \Lambda(w) > \sigma(w)\)
is strict, so equal timestamps count as fresh.

## 6. Policy scope and the gate

A `KickPolicy` selects which workloads it manages and which of their dependency
changes may trigger a restart, via two label selectors:

$$
D(w) = \{\, s \in \mathsf{deps}(w) \ \mid\ \mathsf{labels}(s) \models \text{dependencySelector} \,\},
\qquad
w \text{ managed} \iff \mathsf{labels}(w) \models \text{workloadSelector}.
$$

The **gate** decides whether a permitted-in-principle restart may run *now*:

$$
\mathsf{G} : W \times \mathbb{R}_{\ge 0} \to \{\textsf{Allowed}\} \cup \{\textsf{Blocked}(r) : r \in \mathcal{Q}\},
$$

with blocking reasons \(\mathcal{Q} = \{\textsf{OutsideSchedule},\ \textsf{OwnerUnknown},\ \textsf{MultipleOwners},\ \textsf{OutOfSync},\ \textsf{SyncInProgress},\dots\}\).
It is evaluated in two stages. First, KICK-native schedule windows, if any, are
applied: a set of allow/deny cron windows \(\{(\text{kind}_i,\text{cron}_i,\text{dur}_i)\}\)
induces the predicate

$$
\mathrm{open}(t) \;=\; \Big(\exists i:\ \text{kind}_i=\textsf{Allow}\ \land\ t \in \mathrm{window}_i\Big)\ \land\ \Big(\lnot\exists j:\ \text{kind}_j=\textsf{Deny}\ \land\ t \in \mathrm{window}_j\Big),
$$

and \(\lnot\mathrm{open}(t)\) yields \(\textsf{Blocked}(\textsf{OutsideSchedule})\).
Second, the GitOps provider \(P \in \{\textsf{None},\textsf{Auto},\textsf{ArgoCD},\textsf{Flux}\}\)
is consulted. With \(P=\textsf{None}\) (the default) KICK self-gates and the stage
returns \(\textsf{Allowed}\). Otherwise the owner-resolution relation \(\mathsf{own}(w)\)
must yield exactly one owner whose application is reconciled/synced:

$$
\mathsf{G}(w,t) = \textsf{Allowed} \iff \mathrm{open}(t)\ \land\ \Big(P=\textsf{None}\ \lor\ \big(|\mathsf{own}(w)|=1\ \land\ \mathrm{synced}(\mathsf{own}(w))\big)\Big).
$$

## 7. The restart action

Restarting is the single side effect KICK performs on a workload. It stamps the
Pod template with the standard annotation, which forces the workload controller
to roll a new revision:

$$
\mathsf{A}(w)\ :\quad \mathsf{tmpl}(w).\mathsf{ann}[\textsf{restartedAt}] \;\leftarrow\; \mathrm{now}.
$$

By the definition of \(\sigma\) in §5, immediately after \(\mathsf{A}(w)\) we have
\(\sigma(w) = \mathrm{now} \ge \Lambda(w)\), so the workload is no longer stale.
KICK writes **no** other state: no content hashes, no environment variables, no
owner annotations. The action is the only writer of \(\textsf{restartedAt}\) in
this model.

## 8. The KickRequest transition system

Discovery and coalescing (§9) produce at most one `KickRequest` per target
workload. A request is a state machine over phases

$$
\Phi = \{\textsf{Pending},\ \textsf{WaitingForGate},\ \textsf{WaitingForOwner},\ \textsf{WaitingForApplicationSync},\ \textsf{WaitingForRollout},\ \textsf{Executing},\ \textsf{Succeeded},\ \textsf{NoLongerRequired},\ \textsf{Failed}\},
$$

with terminal set \(\mathsf{Term} = \{\textsf{Succeeded},\ \textsf{NoLongerRequired},\ \textsf{Failed}\}\).
Each request carries a rollout marker \(\mu \in \{\bot\} \cup \mathbb{R}_{\ge 0}\)
(the start time of the rollout it is currently driving). One reconcile step
\(\delta\) applied to a non-terminal request evaluates, in order, the gate, the
freshness relation, and the executor:

$$
\delta(w) =
\begin{cases}
\textsf{WaitingForGate}/\textsf{WaitingForOwner}/\ldots & \text{if } \mathsf{G}(w,t)=\textsf{Blocked}(r),\\[2pt]
\textsf{WaitingForRollout} & \text{if } \mathsf{G}=\textsf{Allowed}\ \land\ \lnot\,\mathrm{complete}(w),\\[2pt]
\textsf{NoLongerRequired} & \text{if } \mathsf{G}=\textsf{Allowed}\ \land\ \mathrm{complete}(w)\ \land\ \lnot\,\mathrm{stale}(w),\\[2pt]
\textsf{Executing} \xrightarrow{\ \mathsf{A}(w)\ } \textsf{Succeeded} & \text{if } \mathsf{G}=\textsf{Allowed}\ \land\ \mathrm{stale}(w).
\end{cases}
$$

The executor issues the physical restart only on the transition into
\(\textsf{Executing}\) when the request has **no** in-flight rollout
(\(\mu=\bot\)); while \(\mu\neq\bot\) it merely watches the rollout to completion.
A terminal request is inert under \(\delta\) except for retention: it is deleted
after a TTL.

## 9. Coalescing and reopening

For a target workload \(w\), the coalescer maintains a single request keyed by
\((\mathsf{ns}(w),\ \mathrm{name}(w),\ \kappa(w))\). On an incoming relevant
change with time \(\theta\):

$$
\mathsf{ensure}(w,\theta):\quad
\begin{cases}
\text{create request in } \textsf{Pending} & \text{if none exists},\\[2pt]
\big(\text{phase} \leftarrow \textsf{Pending},\ \ \mu \leftarrow \bot\big) & \text{if the request is in } \mathsf{Term}\ \ (\textbf{reopen}),\\[2pt]
\text{advance } \textsf{latestObservedDependencyChange} \leftarrow \max(\cdot,\theta) & \text{always.}
\end{cases}
$$

Resetting \(\mu \leftarrow \bot\) on reopen is essential: it is what makes the
next \(\delta\) step run the executor's *start-rollout* path (issuing a fresh
\(\mathsf{A}(w)\)) instead of adopting the already-completed previous rollout.

## 10. Invariants

We collect the properties the operator maintains. Let a *run* be an infinite fair
sequence of reconcile steps under a scheduler that eventually delivers every
enqueued event.

**(I1) No spurious restart at baseline.**
If a source \(s\) is created no later than its consumer's rollout,
\(\gamma(s) \le \sigma(w)\), and no relevant change occurs, then
\(\Lambda(w) \le \sigma(w)\) and hence \(\lnot\,\mathrm{stale}(w)\); by §8 the
request settles in \(\textsf{NoLongerRequired}\) and \(\mathsf{A}(w)\) never
fires. *(Ensured by \(\theta'=\gamma(s)\) on Baseline, §4.)*

**(I2) Eventual restart on relevant change.**
If at time \(t^\star\) a relevant change gives \(\Lambda(w) = t^\star > \sigma(w)\),
and from some time on \(\mathsf{G}(w,\cdot)=\textsf{Allowed}\) and the workload's
rollout is complete, then in every run \(\mathsf{A}(w)\) eventually fires and
afterwards \(\sigma(w) \ge t^\star\). *(Ensured by reopen resetting \(\mu\leftarrow\bot\), §9.)*

**(I3) Kind-agnostic freshness.**
\(\mathrm{stale}(w)\) is well-defined and satisfiable for every
\(\kappa(w) \in \{\textsf{Deployment},\textsf{StatefulSet},\textsf{DaemonSet}\}\),
because completeness is defined per kind and does not require a ReplicaSet.
*(Ensured by the kind-aware \(\mathrm{complete}\), §5.)*

**(I4) At most one active request per target.**
At every reconcile boundary, \(\big|\{\,r : \mathrm{target}(r)=w \ \land\ \mathrm{phase}(r)\notin\mathsf{Term}\,\}\big| \le 1\).
Duplicate and repeated references to the same source therefore cause at most one
concurrent restart. *(Ensured by the keyed coalescer, §9.)*

**(I5) Gate safety.**
\(\mathsf{A}(w)\) fires only from the \(\textsf{Executing}\) transition, which is
reachable only when \(\mathsf{G}(w,t)=\textsf{Allowed}\). Equivalently,

$$
\mathsf{A}(w)\ \text{fires at } t \ \Longrightarrow\ \mathsf{G}(w,t)=\textsf{Allowed}.
$$

**(I6) Idempotence / no rollout amplification.**
After \(\mathsf{A}(w)\) at time \(t'\) with \(t' \ge \Lambda(w)\), we have
\(\sigma(w)=t' \ge \Lambda(w)\), so \(\lnot\,\mathrm{stale}(w)\) and no further
restart is issued until a new relevant change advances \(\Lambda\). A single
change thus yields exactly one new rollout.

**(I7) Non-injection.**
The only workload write is \(\mathsf{A}\), setting the standard
\(\textsf{restartedAt}\) annotation. KICK injects no dependency hashes, no
environment, and no owner state into managed workloads, and never treats
`imagePullSecrets` as dependencies (§2). Secret *values* never appear in
status, events, or logs.

## 11. Reading the pipeline as one relation

Composing the pieces, the end-to-end effect of a relevant change to a source
\(s\) on a managed, in-scope consumer \(w \in \mathsf{cons}(s)\) is

$$
\underbrace{\phi\ \text{changes}}_{\text{observation}}
\ \Rightarrow\
\underbrace{\Lambda(w) > \sigma(w)}_{\text{freshness}}
\ \land\
\underbrace{\mathsf{G}(w,t)=\textsf{Allowed}}_{\text{gate}}
\ \Rightarrow\
\underbrace{\mathsf{A}(w)}_{\text{restart}}.
$$

Everything else the controller does — coalescing, phase transitions, retention —
exists to make this implication hold *exactly once* per change, *only* when
permitted, and *uniformly* across workload kinds.

## See also

- [Freshness](../../concepts/freshness/) — the same idea in prose.
- [GitOps gates](../../concepts/gitops-gates/) — provider and window behaviour.
- [KickPolicy reference](../../reference/kickpolicy/) and
  [KickRequest reference](../../reference/kickrequest/).
