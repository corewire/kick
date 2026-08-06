# Source and consolidation note

These modular specifications were derived from the earlier Reheat concept and the subsequent design decisions in the working conversation. The project name is now **KICK**.

The authoritative implementation requirements are the files under `specs/` and `tasks/`. The earlier monolithic Reheat document is retained outside this bundle only as historical input.


## E2E suite reference repositories

The e2e suite conventions were informed by:

- `corewire/drop`: small behavior-oriented Chainsaw directories, distinct success/failure/discovery scenarios, Kind execution, and a locally built operator image.
- `crossplane-contrib/provider-keycloak`: complete local dependency setup, render-only test inspection, and lifecycle testing that checks stable observation and controller restart/recovery rather than relying on Ready alone.

KICK adopts these structural practices but keeps its own feature IDs, scenario matrix, Argo CD semantics, and security requirements.
