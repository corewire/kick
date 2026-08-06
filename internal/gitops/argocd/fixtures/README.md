# Argo CD adapter fixtures

These fixtures are used by adapter contract tests and parser/gate unit tests.

Files:

- tracking_id_cases.yaml: valid/invalid tracking-id and installation-id cases.
- ownership_fallback_cases.yaml: none/one/many ownership fallback outcomes.
- sync_window_cases.yaml: schedule decisions including deny precedence and selectors.
- rollout_annotation_selfheal_cases.yaml: expected behavior when Argo CD reconciles rollout annotations.
