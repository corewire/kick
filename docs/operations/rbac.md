# RBAC

Controller ClusterRole includes:

- `get/list/watch` on `secrets` and `configmaps`;
- `get/list/watch/patch` on `deployments` and `replicasets`;
- full CRUD on `kickrequests` plus status updates;
- `get/list/watch` on `kickpolicies`;
- lease permissions for leader election.

Source of truth: `config/rbac/role.yaml`.