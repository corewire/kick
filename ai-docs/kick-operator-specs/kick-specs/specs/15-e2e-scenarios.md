# Required end-to-end scenarios

Each scenario MUST have an isolated Chainsaw directory with setup, assertions, stability verification, and cleanup. Follow `20-e2e-suite-conventions.md`.

## Dependency discovery

1. `envFrom` Secret triggers a kick.
2. `env.valueFrom.secretKeyRef` triggers a kick.
3. ConfigMap environment reference triggers a kick.
4. Secret volume triggers a kick.
5. ConfigMap volume triggers a kick.
6. Projected Secret and ConfigMap sources trigger a kick.
7. Init-container references are discovered.
8. `imagePullSecrets` changes do not trigger a kick.
9. Duplicate references create one request.
10. Optional missing Secret creation triggers a kick.

## Change observation

11. Secret data update is relevant.
12. ConfigMap data update is relevant.
13. Metadata-only update is ignored.
14. Reapplying identical content is ignored.
15. Multiple changes coalesce into one active request.
16. Controller restart preserves observations and pending work.

## Freshness

17. Dependency newer than ReplicaSet triggers one rollout.
18. Dependency older than ReplicaSet does not.
19. Dependency removed before action produces `NoLongerRequired`.
20. Normal rollout before action satisfies the request.
21. Active rollout blocks a second rollout.
22. Scaled-to-zero Deployment behaves deterministically.
23. Failed rollout ends in Failed without a restart loop.

## Argo CD ownership

24. Tracking annotation resolves Application in Argo CD namespace.
25. Tracking annotation resolves Application in another namespace.
26. Invalid or mismatched annotation falls back.
27. Fallback index finds exactly one owner.
28. No owner blocks.
29. Multiple owners block.
30. AppProject is read from Argo CD control-plane namespace.
31. Missing AppProject blocks.

## Argo CD gate

32. Open window plus Synced idle Application allows kick.
33. Closed allow window waits.
34. Active deny window waits.
35. Overlapping allow and deny follows Argo CD precedence.
36. AppProject window edit re-enqueues immediately.
37. Application OutOfSync waits.
38. Active sync operation waits.
39. Argo CD sync creates a new ReplicaSet and removes the need for KICK.
40. Argo CD sync changes unrelated resources and KICK still restarts.
41. Synced but Degraded is allowed by default.
42. No applicable windows permits action.

## Restart execution

43. Required kick creates exactly one new ReplicaSet.
44. Reconcile retries do not create a second rollout.
45. Ordinary Argo CD self-heal does not create a second rollout.
46. Pod deletion is never used.
47. Controller leader transition does not duplicate action.

## Recovery

48. Manager restarts while waiting for a window.
49. Manager restarts while waiting for Application sync.
50. Manager restarts after patch and before observing completion.
51. Stale `requeueAt` is safely recomputed.

## Security and framework smoke tests

52. Events, metrics, logs, and statuses never expose Secret data.
53. The Kubebuilder/controller-runtime manager installs through the Helm chart, acquires leadership, exposes health probes, and reconciles a smoke-test resource.

## Traceability

Every scenario MUST have a stable `KICK-E2E-NNN` identifier and a metadata file listing the `KICK-FEAT-NNN` features it proves. Required scenarios may not be silently skipped or disabled.


## Lifecycle expectations

Every scenario that reaches a terminal or stable state MUST prove that the state remains stable across additional reconciliation opportunities. Tests that cause a rollout MUST count ReplicaSets before and after and prove exactly one new ReplicaSet was created. Recovery scenarios MUST restart the KICK manager rather than merely simulating a requeue.
