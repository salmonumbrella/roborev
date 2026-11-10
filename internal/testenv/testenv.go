// Package testenv provides environment isolation helpers for tests.
// This package intentionally has no dependencies on other internal packages
// to avoid import cycles.
package testenv

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
)

// RunIsolatedMain provides a standardized TestMain execution wrapper that
// isolates tests from the production ~/.roborev directory. It safely manages
// environment variables and preserves the original test exit code. Git/Go
// configuration overrides last for the test process lifetime.
func RunIsolatedMain(m *testing.M) int {
	// Snapshot prod log state BEFORE overriding ROBOREV_DATA_DIR.
	// Best-effort: if the home directory can't be resolved (e.g. CI
	// containers with no HOME), skip the barrier rather than aborting.
	var barrier *ProdLogBarrier
	if prodDataDir, err := DefaultProdDataDir(); err == nil {
		barrier = NewProdLogBarrier(prodDataDir)
	}

	tmpDir, cleanupTempDir, err := createIsolatedDataDir()
	if err != nil {
		fmt.Fprintf(os.Stderr, "failed to create temp dir: %v\n", err)
		return 1
	}
	defer cleanupTempDir()

	// TestMain has no caller context before m.Run, so setup owns this root.
	if err := configureGitForTests(context.Background(), tmpDir); err != nil {
		fmt.Fprintf(os.Stderr, "failed to configure test environment: %v\n", err)
		return 1
	}

	restoreDataDir := setDataDirEnv(tmpDir)
	defer restoreDataDir()

	restoreAgentConfigEnv := unsetAgentConfigDirEnv()
	defer restoreAgentConfigEnv()

	code := m.Run()

	// Hard barrier: fail if tests polluted production logs.
	if barrier != nil {
		if msg := barrier.Check(); msg != "" {
			fmt.Fprintln(os.Stderr, msg)
			if code == 0 {
				return 1
			}
		}
	}
	return code
}

// SetDataDir sets ROBOREV_DATA_DIR to a temp directory to isolate tests
// from production ~/.roborev. This is preferred over setting HOME because
// ROBOREV_DATA_DIR takes precedence in config.DataDir(). Returns the temp
// directory path. Cleanup is automatic via t.Setenv.
//
// Note: t.Setenv marks the test as incompatible with t.Parallel(), which is
// appropriate since environment variables are process-global state.
func SetDataDir(t *testing.T) string {
	t.Helper()

	tmpDir := t.TempDir()
	t.Setenv("ROBOREV_DATA_DIR", tmpDir)

	return tmpDir
}

func createIsolatedDataDir() (string, func(), error) {
	tmpDir, err := os.MkdirTemp("", "roborev-test-*")
	if err != nil {
		return "", nil, err
	}
	return tmpDir, func() {
		_ = os.RemoveAll(tmpDir)
	}, nil
}

func setDataDirEnv(dir string) func() {
	origEnv, hasEnv := os.LookupEnv("ROBOREV_DATA_DIR")
	_ = os.Setenv("ROBOREV_DATA_DIR", dir)
	return func() {
		if hasEnv {
			_ = os.Setenv("ROBOREV_DATA_DIR", origEnv)
			return
		}
		_ = os.Unsetenv("ROBOREV_DATA_DIR")
	}
}

// unsetAgentConfigDirEnv clears agent config-dir overrides (e.g.
// CLAUDE_CONFIG_DIR replacing ~/.claude) so tests that redirect HOME never
// resolve paths into a developer's real agent configuration.
func unsetAgentConfigDirEnv() func() {
	keys := []string{"CLAUDE_CONFIG_DIR", "CODEX_HOME", "GROK_HOME", "HERMES_HOME", "COPILOT_HOME", "GEMINI_CLI_HOME", "QWEN_HOME"}
	restores := make([]func(), 0, len(keys))
	for _, key := range keys {
		orig, has := os.LookupEnv(key)
		if !has {
			continue
		}
		_ = os.Unsetenv(key)
		restores = append(restores, func() {
			_ = os.Setenv(key, orig)
		})
	}
	return func() {
		for _, restore := range restores {
			restore()
		}
	}
}

func preserveGoEnvironment(ctx context.Context) error {
	out, err := exec.CommandContext(ctx, "go", "env", "-json", "GOENV", "GOPATH", "GOMODCACHE", "GOCACHE").Output()
	if err != nil {
		return fmt.Errorf("resolve Go environment: %w", err)
	}
	var values map[string]string
	if err := json.Unmarshal(out, &values); err != nil {
		return fmt.Errorf("decode Go environment: %w", err)
	}
	for key, value := range values {
		// GOENV=off resolves to empty; retain the inherited off setting.
		if value != "" {
			if err := os.Setenv(key, value); err != nil {
				return fmt.Errorf("preserve %s: %w", key, err)
			}
		}
	}
	return nil
}

func configureGitForTests(ctx context.Context, tmpDir string) error {
	// Preserve resolved Go settings and cache paths before moving HOME/XDG.
	if err := preserveGoEnvironment(ctx); err != nil {
		return err
	}
	// Git runners that strip GIT_* variables still read HOME/.gitconfig.
	// Isolate the home as well as explicit config overrides so they cannot
	// read persistent user settings after another test restores its HOME.
	_ = os.Setenv("HOME", tmpDir)
	_ = os.Setenv("USERPROFILE", tmpDir)
	// Prevent global/system git config from leaking into tests.
	// Without this, commit.gpgsign=true in global config triggers
	// gpg-agent/pinentry during test commits.
	//
	// Point GIT_CONFIG_GLOBAL at an empty regular file rather than os.DevNull.
	// On Windows os.DevNull is "NUL" (the null device), and Git-for-Windows
	// fails with "fatal: unable to access 'NUL': Invalid argument" when told to
	// read its global config from that device — which broke essentially every
	// git invocation in the suite on Windows (notably Windows-on-ARM). An empty
	// file is a valid, no-op global config on every platform. On Unix
	// /dev/null happens to work, but the empty file is correct everywhere.
	emptyGlobalConfig := filepath.Join(tmpDir, "gitconfig-global")
	if f, err := os.OpenFile(emptyGlobalConfig, os.O_CREATE|os.O_WRONLY, 0o644); err == nil {
		_ = f.Close()
	}
	_ = os.Setenv("GIT_CONFIG_GLOBAL", emptyGlobalConfig)
	_ = os.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	// Git also reads the default global ignore file from XDG_CONFIG_HOME
	// independently of GIT_CONFIG_GLOBAL. Keep that out of the user's real
	// ~/.config so personal excludes cannot change test behavior.
	xdgConfigHome := filepath.Join(tmpDir, "xdg-config")
	_ = os.MkdirAll(xdgConfigHome, 0o755)
	_ = os.Setenv("XDG_CONFIG_HOME", xdgConfigHome)
	// Never allow git to prompt for input (passwords, passphrases, etc).
	// If something unexpected tries to prompt, fail fast instead of blocking.
	_ = os.Setenv("GIT_TERMINAL_PROMPT", "0")
	return nil
}
