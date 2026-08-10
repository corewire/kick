# Scalability

Initial implementation targets correctness over horizontal scale optimization.

Current characteristics:

- one KickRequest per target Deployment;
- request coalescing prevents duplicate restarts for the same target;
- gate and freshness checks are recomputed from live state before action.

Operational guidance:

- monitor `kick_controller_errors_total` and queue behavior;
- scale controller resources before increasing managed namespace count;
- validate provider API rate behavior in your environment.