# Security

KICK requires read access to Secrets and ConfigMaps in managed namespaces to evaluate dependency freshness.

Implications:

- treat controller ServiceAccount as sensitive;
- restrict namespace scope where possible;
- avoid exposing logs broadly.

KICK safety constraints:

- no privileged containers;
- no CRI socket access;
- no Secret value logging.