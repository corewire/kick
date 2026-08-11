# Feature Coverage Report

| Feature | Unit | Envtest | E2E | Required E2E scenarios | Result |
|---|---:|---:|---:|---|---|
| KICK-FEAT-001 | Pass | Pass | Pass | KICK-E2E-001, KICK-E2E-002, KICK-E2E-003, KICK-E2E-004, KICK-E2E-005, KICK-E2E-006, KICK-E2E-007, KICK-E2E-010, KICK-E2E-054, KICK-E2E-055 | PASS |
| KICK-FEAT-002 | Pass | Optional | Pass | KICK-E2E-001, KICK-E2E-002, KICK-E2E-003, KICK-E2E-007 | PASS |
| KICK-FEAT-003 | Pass | Optional | Pass | KICK-E2E-004, KICK-E2E-005, KICK-E2E-006 | PASS |
| KICK-FEAT-004 | Pass | Optional | Pass | KICK-E2E-008 | PASS |
| KICK-FEAT-005 | Pass | Pass | Pass | KICK-E2E-011, KICK-E2E-012, KICK-E2E-013, KICK-E2E-014 | PASS |
| KICK-FEAT-006 | Pass | Pass | Pass | KICK-E2E-016, KICK-E2E-048, KICK-E2E-049, KICK-E2E-050, KICK-E2E-051 | PASS |
| KICK-FEAT-007 | Pass | Pass | Pass | KICK-E2E-017, KICK-E2E-018, KICK-E2E-019, KICK-E2E-020, KICK-E2E-021, KICK-E2E-022, KICK-E2E-023, KICK-E2E-054, KICK-E2E-055 | PASS |
| KICK-FEAT-008 | Pass | Optional | Pass | KICK-E2E-032, KICK-E2E-037, KICK-E2E-038 | PASS |
| KICK-FEAT-009 | Pass | Pass | Pass | KICK-E2E-024, KICK-E2E-025, KICK-E2E-026 | PASS |
| KICK-FEAT-010 | Pass | Pass | Pass | KICK-E2E-027, KICK-E2E-028, KICK-E2E-029 | PASS |
| KICK-FEAT-011 | Pass | Pass | Pass | KICK-E2E-030, KICK-E2E-031 | PASS |
| KICK-FEAT-012 | Pass | Pass | Pass | KICK-E2E-032, KICK-E2E-033, KICK-E2E-034, KICK-E2E-035, KICK-E2E-036, KICK-E2E-042 | PASS |
| KICK-FEAT-013 | Pass | Pass | Pass | KICK-E2E-037, KICK-E2E-038, KICK-E2E-039, KICK-E2E-040, KICK-E2E-041 | PASS |
| KICK-FEAT-014 | Pass | Pass | Pass | KICK-E2E-009, KICK-E2E-015, KICK-E2E-048, KICK-E2E-049, KICK-E2E-050 | PASS |
| KICK-FEAT-015 | Pass | Pass | Pass | KICK-E2E-043, KICK-E2E-044, KICK-E2E-045, KICK-E2E-046, KICK-E2E-047 | PASS |
| KICK-FEAT-016 | Pass | Pass | Pass | KICK-E2E-052 | PASS |
| KICK-FEAT-017 | N/A | Pass | Pass | KICK-E2E-053 | PASS |
| KICK-FEAT-018 | Pass | N/A | N/A | - | PASS |
| KICK-FEAT-019 | Pass | N/A | N/A | - | PASS |
| KICK-FEAT-020 | Pass | Pass | Pass | KICK-E2E-056 | PASS |
| KICK-FEAT-021 | Pass | Optional | Pass | KICK-E2E-057 | PASS |
| KICK-FEAT-022 | Pass | Optional | Pass | KICK-E2E-058 | PASS |
| KICK-FEAT-023 | Pass | N/A | N/A | - | PASS |
| KICK-FEAT-024 | Pass | N/A | Pass | KICK-E2E-060, KICK-E2E-061, KICK-E2E-062, KICK-E2E-063 | PASS |
| KICK-FEAT-025 | Pass | N/A | N/A | - | PASS |
| KICK-FEAT-026 | Pass | Pass | Pass | KICK-E2E-059 | PASS |
| KICK-FEAT-027 | Pass | Pass | Pass | KICK-E2E-072 | PASS |

# API Field Coverage Report

| Type | Field | Required | Unit | Envtest | E2E | Result | Evidence |
|---|---|---:|---:|---:|---:|---|---|
| KickRequestSpec | targetRef.apiVersion | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickrequest_api_test.go |
| KickRequestSpec | targetRef.kind | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickrequest_api_test.go |
| KickRequestSpec | targetRef.name | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickrequest_api_test.go |
| KickRequestSpec | policyRef.name | yes | Covered | Missing | Missing | PASS | unit=internal/kickrequest/coalescer_test.go |
| KickRequestStatus | phase | yes | Covered | Covered | Covered | PASS | unit=internal/kickrequest/coalescer_test.go,internal/controller/kickrequest_controller_test.go; envtest=test/envtest/kickrequest_api_test.go; e2e=KICK-E2E-009(covered),KICK-E2E-015(covered),KICK-E2E-048(covered) |
| KickRequestStatus | owner.provider | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickrequest_api_test.go |
| KickRequestStatus | owner.apiVersion | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickrequest_api_test.go |
| KickRequestStatus | owner.kind | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickrequest_api_test.go |
| KickRequestStatus | owner.namespace | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickrequest_api_test.go |
| KickRequestStatus | owner.name | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickrequest_api_test.go |
| KickRequestStatus | owner.project | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickrequest_api_test.go |
| KickRequestStatus | gate.reason | yes | Covered | Covered | Covered | PASS | unit=internal/controller/kickrequest_controller_test.go; envtest=test/envtest/kickrequest_api_test.go; e2e=KICK-E2E-033(covered),KICK-E2E-034(covered) |
| KickRequestStatus | gate.message | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickrequest_api_test.go |
| KickRequestStatus | gate.requeueAt | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickrequest_api_test.go |
| KickRequestStatus | latestObservedDependencyChange | yes | Covered | Covered | Covered | PASS | unit=internal/kickrequest/coalescer_test.go; envtest=test/envtest/kickrequest_api_test.go; e2e=KICK-E2E-011(covered),KICK-E2E-012(covered) |
| KickRequestStatus | currentRollout.replicaSet | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickrequest_api_test.go |
| KickRequestStatus | currentRollout.startedAt | yes | Covered | Covered | Covered | PASS | unit=internal/executor/restart_test.go,internal/controller/kickrequest_controller_test.go; envtest=test/envtest/kickrequest_api_test.go; e2e=KICK-E2E-043(covered) |
| KickRequestStatus | conditions | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickrequest_api_test.go |
| KickPolicySpec | suspend | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickpolicy_api_test.go |
| KickPolicySpec | dryRun | yes | Covered | Missing | Missing | PASS | unit=internal/controller/dryrun_test.go |
| KickPolicySpec | discovery.workloadSelector | yes | Covered | Covered | Missing | PASS | unit=internal/policy/matcher_test.go; envtest=test/envtest/kickpolicy_api_test.go |
| KickPolicySpec | discovery.dependencySelector | yes | Covered | Covered | Missing | PASS | unit=internal/controller/kickrequest_controller_test.go,internal/controller/kickrequest_enqueuer_test.go; envtest=test/envtest/kickpolicy_api_test.go |
| KickPolicySpec | schedule.windows.type | yes | Missing | Missing | Covered | PASS | e2e=KICK-E2E-015(covered),KICK-E2E-036(covered),KICK-E2E-048(covered) |
| KickPolicySpec | schedule.windows.cron | yes | Missing | Missing | Covered | PASS | e2e=KICK-E2E-015(covered),KICK-E2E-036(covered),KICK-E2E-048(covered) |
| KickPolicySpec | schedule.windows.duration | yes | Missing | Missing | Covered | PASS | e2e=KICK-E2E-015(covered),KICK-E2E-036(covered),KICK-E2E-048(covered) |
| KickPolicySpec | schedule.windows.timeZone | yes | Missing | Missing | Covered | PASS | e2e=KICK-E2E-015(covered),KICK-E2E-036(covered),KICK-E2E-048(covered) |
| KickPolicySpec | gitOps.provider | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickpolicy_api_test.go |
| KickPolicySpec | gitOps.requireReconciled | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickpolicy_api_test.go |
| KickPolicySpec | restart.minInterval | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickpolicy_api_test.go |
| KickPolicyStatus | observedGeneration | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickpolicy_api_test.go |
| KickPolicyStatus | matchedWorkloads | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickpolicy_api_test.go |
| KickPolicyStatus | blockedWorkloads | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickpolicy_api_test.go |
| KickPolicyStatus | conditions | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/kickpolicy_api_test.go |
| NotificationPolicySpec | suspend | yes | Covered | Covered | Missing | PASS | unit=internal/notify/dispatcher_test.go; envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | phases | yes | Covered | Covered | Missing | PASS | unit=internal/notify/dispatcher_test.go; envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | workloadSelector | yes | Covered | Covered | Missing | PASS | unit=internal/notify/dispatcher_test.go; envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | webhook.url | yes | Covered | Covered | Missing | PASS | unit=internal/notify/dispatcher_test.go; envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | webhook.method | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | webhook.timeoutSeconds | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | webhook.headers.name | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | webhook.headers.value | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | webhook.headers.valueFrom.name | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | webhook.headers.valueFrom.key | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | webhook.auth.bearerToken.name | yes | Covered | Covered | Missing | PASS | unit=internal/notify/dispatcher_test.go; envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | webhook.auth.bearerToken.key | yes | Covered | Covered | Missing | PASS | unit=internal/notify/dispatcher_test.go; envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | webhook.auth.basic.username.name | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | webhook.auth.basic.username.key | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | webhook.auth.basic.password.name | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | webhook.auth.basic.password.key | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | webhook.tls.caBundle.name | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | webhook.tls.caBundle.key | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | webhook.tls.clientCertificate.name | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicySpec | webhook.tls.clientCertificate.key | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicyStatus | observedGeneration | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicyStatus | lastDeliveryTime | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicyStatus | lastError | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicyStatus | delivered | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicyStatus | failed | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
| NotificationPolicyStatus | conditions | yes | Missing | Covered | Missing | PASS | envtest=test/envtest/notificationpolicy_api_test.go |
