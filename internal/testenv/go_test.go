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
	for _, name := range []string{"defaults", "persisted settings and environment override", "GOENV off"} {
		t.Run(name, func(t *testing.T) {
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
			switch name {
			case "persisted settings and environment override":
				configPath := readGoEnv()["GOENV"]
				rel, err := filepath.Rel(root, configPath)
				require.NoError(t, err)
				require.True(t, filepath.IsLocal(rel), "Go settings must stay inside the fixture")
				require.NoError(t, os.MkdirAll(filepath.Dir(configPath), 0o755))
				settings := "GOMODCACHE=" + filepath.Join(root, "persisted-modules") + "\nGOFLAGS=-mod=readonly\nGOCACHE=" + filepath.Join(root, "persisted-cache") + "\n"
				require.NoError(t, os.WriteFile(configPath, []byte(settings), 0o600))
				t.Setenv("GOCACHE", filepath.Join(root, "explicit-cache"))
			case "GOENV off":
				t.Setenv("GOENV", "off")
			}
			before := readGoEnv()
			if name == "persisted settings and environment override" {
				require.Equal(t, "-mod=readonly", before["GOFLAGS"])
				require.Equal(t, filepath.Join(root, "persisted-modules"), before["GOMODCACHE"])
				require.Equal(t, filepath.Join(root, "explicit-cache"), before["GOCACHE"])
			}
			require.NoError(t, configureGitForTests(t.Context(), t.TempDir()))
			assert.Equal(t, before, readGoEnv())
			if name == "GOENV off" {
				assert.Equal(t, "off", os.Getenv("GOENV"))
			}
		})
	}
}
