# 
# External Secrets

KICK reacts to resulting Kubernetes Secret and ConfigMap changes, not to external secret provider events directly.

Recommended pattern:

1. external system updates secret source;
2. sync controller writes Kubernetes Secret/ConfigMap;
3. KICK observes content change and evaluates freshness.

Notes:

- metadata-only updates are ignored;
- unchanged data values are ignored;
- Secret read RBAC is required in target namespaces.
