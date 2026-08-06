package argocd

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

type windowFixtureFile struct {
	Cases []windowFixture `yaml:"cases"`
}

type windowFixture struct {
	ID      string                   `yaml:"id"`
	Windows []map[string]interface{} `yaml:"windows"`
	App     struct {
		Name              string `yaml:"name"`
		DestinationNS     string `yaml:"destinationNamespace"`
		DestinationName   string `yaml:"destinationClusterName"`
		DestinationServer string `yaml:"destinationServer"`
	} `yaml:"app"`
	Now    time.Time `yaml:"now"`
	Expect struct {
		Allowed *bool  `yaml:"allowed"`
		Matched *bool  `yaml:"matched"`
		Reason  string `yaml:"reason"`
	} `yaml:"expect"`
}

func TestSyncWindowCompatibilityFixtures(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("fixtures", "sync_window_cases.yaml"))
	if err != nil {
		t.Fatalf("read fixture: %v", err)
	}
	var fixture windowFixtureFile
	if err := yaml.Unmarshal(data, &fixture); err != nil {
		t.Fatalf("unmarshal fixture: %v", err)
	}

	for _, c := range fixture.Cases {
		t.Run(c.ID, func(t *testing.T) {
			ctx := appWindowContext{Name: c.App.Name, DestinationNS: c.App.DestinationNS, DestinationName: c.App.DestinationName, DestinationServer: c.App.DestinationServer}
			result, err := evaluateSyncWindows(c.Now, ctx, c.Windows)
			if err != nil {
				t.Fatalf("evaluate: %v", err)
			}
			if c.Expect.Allowed != nil && result.Allowed != *c.Expect.Allowed {
				t.Fatalf("allowed=%v want %v", result.Allowed, *c.Expect.Allowed)
			}
			if c.Expect.Reason != "" && string(result.Reason) != c.Expect.Reason {
				t.Fatalf("reason=%s want %s", result.Reason, c.Expect.Reason)
			}
			if c.Expect.Matched != nil {
				matched := matchesAnyWindow(ctx, c.Windows)
				if matched != *c.Expect.Matched {
					t.Fatalf("matched=%v want %v", matched, *c.Expect.Matched)
				}
			}
		})
	}
}

func matchesAnyWindow(ctx appWindowContext, windows []map[string]interface{}) bool {
	for _, w := range windows {
		if matchesWindow(ctx, w) {
			return true
		}
	}
	return false
}
