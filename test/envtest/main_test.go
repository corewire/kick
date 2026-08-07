package envtest

import (
	"os"
	"path/filepath"
	"sort"
	"testing"
)

func TestMain(m *testing.M) {
	if os.Getenv("KUBEBUILDER_ASSETS") == "" {
		if assets := findLocalEnvtestAssets(); assets != "" {
			_ = os.Setenv("KUBEBUILDER_ASSETS", assets)
		}
	}

	os.Exit(m.Run())
}

func findLocalEnvtestAssets() string {
	wd, err := os.Getwd()
	if err != nil {
		return ""
	}

	repoRoot := filepath.Clean(filepath.Join(wd, "..", ".."))
	matches, err := filepath.Glob(filepath.Join(repoRoot, "bin", "k8s", "*"))
	if err != nil {
		return ""
	}
	sort.Strings(matches)

	for index := len(matches) - 1; index >= 0; index-- {
		candidate := matches[index]
		if fileExists(filepath.Join(candidate, "etcd")) && fileExists(filepath.Join(candidate, "kube-apiserver")) {
			return candidate
		}
	}

	return ""
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}