package envtest

import (
	"path/filepath"
	"testing"

	kickv1alpha1 "github.com/corewire/kick/api/v1alpha1"
	"k8s.io/apimachinery/pkg/runtime"
	utilruntime "k8s.io/apimachinery/pkg/util/runtime"
	"sigs.k8s.io/controller-runtime/pkg/envtest"
)

// TestEnvtestBoot starts a real API server with local CRDs to validate the
// controller-runtime/Kubebuilder test foundation wiring.
func TestEnvtestBoot(t *testing.T) {
	t.Parallel()

	scheme := runtime.NewScheme()
	utilruntime.Must(kickv1alpha1.AddToScheme(scheme))

	env := &envtest.Environment{
		CRDDirectoryPaths: []string{filepath.Join("..", "..", "config", "crd", "bases")},
	}

	cfg, err := env.Start()
	if err != nil {
		t.Fatalf("start envtest: %v", err)
	}
	if cfg == nil {
		t.Fatalf("envtest returned nil rest config")
	}

	if err := env.Stop(); err != nil {
		t.Fatalf("stop envtest: %v", err)
	}
}
