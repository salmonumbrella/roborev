package main

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"go.kenn.io/roborev/internal/daemon"
)

func TestUICmdOpensReviewRoutes(t *testing.T) {
	tests := []struct {
		name string
		args []string
		want string
	}{
		{name: "reviews", want: "https://reviews.example.com/reviews"},
		{name: "job", args: []string{"42"}, want: "https://reviews.example.com/reviews/42"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var opened string
			withUICommandDependencies(t,
				func() error { return nil },
				func() (*daemon.RuntimeInfo, error) {
					return &daemon.RuntimeInfo{WebOrigin: "https://reviews.example.com"}, nil
				},
				func(target string) error { opened = target; return nil },
			)
			cmd := uiCmd()
			cmd.SetArgs(tt.args)

			require.NoError(t, cmd.Execute())
			assert.Equal(t, tt.want, opened)
			assert.NotContains(t, opened, "configured-secret")
		})
	}
}

func TestUICmdRejectsInvalidJobIDs(t *testing.T) {
	for _, args := range [][]string{{"0"}, {"-1"}, {"nope"}, {"1", "2"}} {
		cmd := uiCmd()
		cmd.SetArgs(args)
		require.Error(t, cmd.Execute())
	}
}

func TestUICmdRequiresBrowserMetadata(t *testing.T) {
	withUICommandDependencies(t,
		func() error { return nil },
		func() (*daemon.RuntimeInfo, error) { return &daemon.RuntimeInfo{}, nil },
		func(string) error { return nil },
	)
	cmd := uiCmd()
	err := cmd.Execute()
	require.EqualError(t, err, "the daemon browser listener is disabled")
}

func TestUICmdPreservesDiscoveryAccessDenial(t *testing.T) {
	want := daemon.ErrDaemonAccessDenied
	withUICommandDependencies(t,
		func() error { return want },
		func() (*daemon.RuntimeInfo, error) { return nil, errors.New("unexpected discovery") },
		func(string) error { return nil },
	)
	cmd := uiCmd()
	err := cmd.Execute()
	require.ErrorIs(t, err, want)
}

func TestUICmdOpenerErrorIncludesManualURL(t *testing.T) {
	withUICommandDependencies(t,
		func() error { return nil },
		func() (*daemon.RuntimeInfo, error) {
			return &daemon.RuntimeInfo{WebOrigin: "https://reviews.example.com"}, nil
		},
		func(string) error { return errors.New("no opener") },
	)
	cmd := uiCmd()
	cmd.SetArgs([]string{"7"})
	err := cmd.Execute()
	require.Error(t, err)
	assert.Contains(t, err.Error(), "https://reviews.example.com/reviews/7")
}

func TestUICmdRejectsInvalidPublishedOrigin(t *testing.T) {
	for _, origin := range []string{
		"file:///tmp/reviews",
		"https://user@example.com",
		"https://reviews.example.com/path",
		"https://reviews.example.com?token=secret",
	} {
		withUICommandDependencies(t,
			func() error { return nil },
			func() (*daemon.RuntimeInfo, error) {
				return &daemon.RuntimeInfo{WebOrigin: origin}, nil
			},
			func(string) error { return nil },
		)
		cmd := uiCmd()
		require.Error(t, cmd.Execute(), origin)
	}
}

func withUICommandDependencies(
	t *testing.T,
	ensure func() error,
	discover func() (*daemon.RuntimeInfo, error),
	open func(string) error,
) {
	t.Helper()
	originalEnsure := uiEnsureDaemon
	originalDiscover := uiGetAnyRunningDaemon
	originalOpen := openBrowserURL
	uiEnsureDaemon = ensure
	uiGetAnyRunningDaemon = discover
	openBrowserURL = open
	t.Cleanup(func() {
		uiEnsureDaemon = originalEnsure
		uiGetAnyRunningDaemon = originalDiscover
		openBrowserURL = originalOpen
	})
}
