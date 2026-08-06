KICK_CONTEXT = 'kind-kick-dev'
KICK_KUBECONFIG = os.getenv('KICK_KUBECONFIG', '.kubeconfig-kind-kick-dev')

allow_k8s_contexts(KICK_CONTEXT)

if os.getenv('KUBECONFIG', '') not in ['', KICK_KUBECONFIG]:
    fail('KUBECONFIG must point to .kubeconfig-kind-kick-dev for kick local dev')

if k8s_context() != KICK_CONTEXT:
    fail('Current kube context must be kind-kick-dev')

local_resource(
    'preflight',
    cmd='make tools manifests',
)

docker_build(
    'ghcr.io/corewire/kick',
    '.',
    dockerfile='Dockerfile',
    only=['api', 'cmd', 'config', 'internal', 'go.mod', 'go.sum'],
)

k8s_yaml(kustomize('config/default'))

k8s_resource(
    'kick-controller-manager',
    port_forwards=['8080:8080', '8081:8081'],
    resource_deps=['preflight'],
)
