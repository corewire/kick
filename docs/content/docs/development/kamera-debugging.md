# Kamera Debugging for KICK

KICK includes optional kamera tooling for structural control-plane analysis and exploration artifact inspection.

Upstream: https://github.com/tgoodwin/kamera

## Install

From repo root:

```bash
make kamera
bin/kamera --help
```

The Make target installs:

```bash
go install github.com/tgoodwin/kamera/cmd/kamera@main
```

## KICK example you can run now

This repository now includes a concrete KICK graph example with real KICK resource kinds:

- docs/development/examples/kamera/kick-dependency-graph.json

It models:

- SourceObservationReconciler
- KickRequestReconciler
- core/v1 Secret + ConfigMap
- apps/v1 Deployment
- kick.corewire.io/v1alpha1 KickRequest

### 1) Detect hotspots in the KICK graph

```bash
bin/kamera inspect hotspots docs/development/examples/kamera/kick-dependency-graph.json
```

Expected hotspot types in output:

- multi_writer on KickRequest status
- feedback_cycle on KickRequestReconciler <-> KickRequest
- reducer_controller on KickRequestReconciler
- missing_trigger on resources KickRequestReconciler reads but does not directly watch

This gives a fast structural risk scan before runtime debugging.

### 2) Render dependency graph PDF

```bash
bin/kamera inspect dependency-graph docs/development/examples/kamera/kick-dependency-graph.json
```

On Linux this uses xdg-open and reports a temp PDF path such as:

```text
opened dependency graph from docs/development/examples/kamera/kick-dependency-graph.json
pdf saved at /tmp/dependency-graph-<id>.pdf
```

Use this graph during design reviews for controller/resource coupling.

## Exploration dumps (when you have one)

If you have an exploration dump file or directory from a kamera harness:

```bash
bin/kamera inspect exploration <dump.jsonl>
bin/kamera inspect exploration --interactive=false <dump.jsonl>
bin/kamera analyze report <dump.jsonl>
bin/kamera analyze diff <dump.jsonl>
```

For headless CI triage, prefer:

```bash
bin/kamera inspect exploration --interactive=false <dump.jsonl>
```

This prints DAG + node details directly to stdout.

## Determinize helper

Kamera also provides deterministic time rewrite tooling:

```bash
bin/kamera determinize [paths...]
```

Use this on isolated experiments, not as a blanket rewrite of the repository.

## KICK workflow recommendation

1. Reproduce issue with existing checks (`make test`, `make test-e2e-scenario E2E=...`).
2. Run hotspot scan on a graph snapshot of the affected controllers/resources.
3. Inspect exploration dumps (if available) to isolate ordering-sensitive paths.
4. Convert findings into deterministic unit/envtest/e2e assertions in KICK.

## Notes

- kamera generate currently depends on upstream translation code paths that are evolving; prefer inspect/analyze workflows for now.
- kamera tagged module versions currently lag cmd/kamera availability, so this repository pins installation to main.
