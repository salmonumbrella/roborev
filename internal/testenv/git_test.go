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
	for _, key := range []string{"HOME", "USERPROFILE", "GIT_CONFIG_GLOBAL", "GIT_CONFIG_NOSYSTEM", "XDG_CONFIG_HOME", "GIT_TERMINAL_PROMPT"} {
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
	configureGitForTests(t.TempDir())
	assert.Empty(t, readGlobalConfig(), "stripping GIT_* must not expose the caller's persistent config")

	// A test may install its own global config, then restore HOME on cleanup.
	// Later tests must return to the isolated home, not the caller's config.
	t.Run("temporary global config", func(t *testing.T) {
		fixtureHome := t.TempDir()
		t.Setenv("HOME", fixtureHome)
		t.Setenv("USERPROFILE", fixtureHome)
		require.NoError(t, os.WriteFile(filepath.Join(fixtureHome, ".gitconfig"), []byte("[core]\n\thooksPath = fixture-hooks\n"), 0o600))
		assert.Equal(t, "core.hookspath=fixture-hooks", readGlobalConfig())
	})
	assert.Empty(t, readGlobalConfig(), "global config from an earlier test must not affect later tests")
	data, err := os.ReadFile(configPath)
	require.NoError(t, err)
	assert.Equal(t, config, data, "the caller's config must remain unchanged")
}
