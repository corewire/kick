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

k8s_yaml(kustomize('config/dev'))

# --- Tracing backend: Jaeger all-in-one (OTLP receiver + UI) ---
k8s_yaml('hack/tracing/jaeger.yaml')

k8s_resource(
    'jaeger',
    port_forwards=['16686:16686'],
    links=['http://localhost:16686/'],
    labels=['tracing'],
)

k8s_resource(
    'kick-controller-manager',
    resource_deps=['preflight', 'jaeger'],
    port_forwards=['8090:8090'],
)

# --- Documentation: Hugo Hextra (live reload) ---
local_resource(
    'docs',
    serve_cmd='cd docs && hugo server --buildDrafts --port 1313 --bind 0.0.0.0',
    deps=['docs/content', 'docs/hugo.yaml', 'docs/assets', 'docs/static', 'docs/layouts'],
    links=['http://localhost:1313/kick/'],
    labels=['docs'],
)
