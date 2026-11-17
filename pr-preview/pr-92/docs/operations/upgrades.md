# 
# Upgrades

API maturity: `v1alpha1`.

Policy:

- breaking changes may occur before beta;
- when production users persist objects, migrations or conversion guidance must be documented.

Upgrade flow:

1. review release notes;
2. apply updated CRDs;
3. upgrade chart image/tag;
4. watch controller readiness and KickRequest reconciliation.
