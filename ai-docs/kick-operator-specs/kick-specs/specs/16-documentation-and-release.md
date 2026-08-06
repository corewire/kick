# Documentation and release requirements

## Documentation structure

```text
docs/
  getting-started/
    installation.md
    quickstart.md
  concepts/
    hidden-restart-requirement.md
    dependency-discovery.md
    freshness.md
    gitops-gates.md
  guides/
    external-secrets.md
    argocd.md
    troubleshooting.md
  reference/
    kickpolicy.md
    kickrequest.md
    configuration.md
    metrics.md
    events.md
  operations/
    security.md
    rbac.md
    upgrades.md
    scalability.md
  decisions/
```

## Required explanations

Documentation MUST explain:

- why Kubernetes does not restart consumers automatically;
- environment variables versus mounted-file behavior;
- what KICK considers a dependency;
- why imagePullSecrets are excluded;
- how KICK determines stale versus fresh;
- initial-baseline limitations;
- Argo CD ownership discovery;
- sync-window and Synced-state gating;
- fields KICK mutates;
- Secret-read privilege and security implications;
- failure and recovery behavior.

## Generated documentation

Generate and freshness-check:

- CRD API reference;
- defaults and validation;
- CLI reference if a CLI exists;
- metrics and event reference;
- example manifests;
- `llms.txt` and `llms-full.txt`;
- `AGENTS.md` or project-specific coding instructions.

Follow DROP's pattern of `docs-gen` plus `docs-gen-check`.

## Release artifacts

Publish:

- signed container images;
- Helm chart;
- CRD bundle;
- SBOM;
- checksums;
- release notes with supported Kubernetes and Argo CD versions.

## Versioning

The first API version is `v1alpha1`. Breaking API changes are permitted before beta but require conversion or documented migration once users can persist production objects.

## Acceptance criteria

- A new user can install KICK and complete the Argo CD quickstart.
- Generated docs cannot drift unnoticed.
- Security and compatibility limitations are explicit.
