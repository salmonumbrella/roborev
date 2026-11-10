package testenv

import (
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestConfigureGitForTestsPreservesGoEnvironment(t *testing.T) {
	tests := []struct {
		name       string
		setup      func(t *testing.T, root string, readGoEnv func() map[string]string)
		wantBefore func(root string) map[string]string
		wantGOENV  string
	}{
		{name: "defaults"},
		{
			name: "persisted settings and environment override",
			setup: func(t *testing.T, root string, readGoEnv func() map[string]string) {
				configPath := readGoEnv()["GOENV"]
				rel, err := filepath.Rel(root, configPath)
				require.NoError(t, err)
				require.True(t, filepath.IsLocal(rel), "Go settings must stay inside the fixture")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0o755))
				settings := "GOMODCACHE=" + filepath.Join(root, "persisted-modules") + "\nGOFLAGS=-mod=readonly\nGOCACHE=" + filepath.Join(root, "persisted-cache") + "\n"
				require.NoError(t, os.WriteFile(configPath, []byte(settings), 0o600))
				t.Setenv("GOCACHE", filepath.Join(root, "explicit-cache"))
			},
			wantBefore: func(root string) map[string]string {
				return map[string]string{
					"GOFLAGS":    "-mod=readonly",
					"GOMODCACHE": filepath.Join(root, "persisted-modules"),
					"GOCACHE":    filepath.Join(root, "explicit-cache"),
				}
			},
		},
		{
			name: "GOENV off",
			setup: func(t *testing.T, _ string, _ func() map[string]string) {
				t.Setenv("GOENV", "off")
			},
			wantGOENV: "off",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := t.TempDir()
			t.Chdir(root)
			for _, key := range []string{"HOME", "USERPROFILE", "XDG_CONFIG_HOME", "APPDATA", "LOCALAPPDATA"} {
				dir := filepath.Join(root, key)
				require.NoError(t, os.MkdirAll(dir, 0o755))
				t.Setenv(key, dir)
			}
			for _, key := range []string{"GOENV", "GOPATH", "GOMODCACHE", "GOCACHE", "GOFLAGS", "XDG_CACHE_HOME"} {
				t.Setenv(key, "")
				require.NoError(t, os.Unsetenv(key))
			}
			for _, key := range []string{"GIT_CONFIG_GLOBAL", "GIT_CONFIG_NOSYSTEM", "GIT_TERMINAL_PROMPT"} {
				t.Setenv(key, os.Getenv(key))
			}
			t.Setenv("GOTOOLCHAIN", "local")
			readGoEnv := func() map[string]string {
				t.Helper()
				cmd := exec.CommandContext(t.Context(), "go", "env", "-json", "GOENV", "GOPATH", "GOMODCACHE", "GOCACHE", "GOFLAGS")
				out, err := cmd.Output()
				require.NoError(t, err)
				var values map[string]string
				require.NoError(t, json.Unmarshal(out, &values))
				return values
			}
			if tt.setup != nil {
				tt.setup(t, root, readGoEnv)
			}
			before := readGoEnv()
			if tt.wantBefore != nil {
				for key, value := range tt.wantBefore(root) {
					require.Equal(t, value, before[key])
				}
			}
			require.NoError(t, configureGitForTests(t.Context(), t.TempDir()))
			assert.Equal(t, before, readGoEnv())
			if tt.wantGOENV != "" {
				assert.Equal(t, tt.wantGOENV, os.Getenv("GOENV"))
			}
		})
	}
}
