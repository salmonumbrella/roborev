package testenv

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	gitcmd "go.kenn.io/kit/git/cmd"
)

func TestConfigureGitForTestsIsolatesPersistentConfig(t *testing.T) {
	// Restore every variable configureGitForTests changes after this test.
	for _, key := range []string{"HOME", "USERPROFILE", "GIT_CONFIG_GLOBAL", "GIT_CONFIG_NOSYSTEM", "XDG_CONFIG_HOME", "GIT_TERMINAL_PROMPT", "GOENV", "GOPATH", "GOMODCACHE", "GOCACHE"} {
		t.Setenv(key, os.Getenv(key))
	}
	home := t.TempDir()
	t.Setenv("HOME", home)
	t.Setenv("USERPROFILE", home)
	config := []byte("[core]\n\thooksPath = inherited-hooks\n")
	configPath := filepath.Join(home, ".gitconfig")
	require.NoError(t, os.WriteFile(configPath, config, 0o600))

	runner := gitcmd.Runner{StripEnv: true, DisableSafeDirectoryForward: true}
	readGlobalConfig := func() string {
		t.Helper()
		cmd := runner.Command(t.Context(), home, "config", "--global", "--list")
		out, err := cmd.Output()
		require.NoError(t, err)
		return strings.TrimSpace(string(out))
	}
	require.Equal(t, "core.hookspath=inherited-hooks", readGlobalConfig(), "control must read the persistent fixture")
	require.NoError(t, configureGitForTests(t.Context(), t.TempDir()))
	assert.Empty(t, readGlobalConfig(), "stripping GIT_* must not expose the caller's persistent config")
}
