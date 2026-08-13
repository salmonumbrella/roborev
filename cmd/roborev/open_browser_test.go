package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenBrowserUsesPlatformCommandWithoutShell(t *testing.T) {
	tests := []struct {
		goos     string
		wantName string
		wantArgs []string
	}{
		{goos: "darwin", wantName: "open", wantArgs: []string{"https://example.com/reviews"}},
		{goos: "windows", wantName: "rundll32", wantArgs: []string{"url.dll,FileProtocolHandler", "https://example.com/reviews"}},
		{goos: "linux", wantName: "xdg-open", wantArgs: []string{"https://example.com/reviews"}},
	}
	for _, tt := range tests {
		t.Run(tt.goos, func(t *testing.T) {
			originalGOOS := browserGOOS
			originalStart := startBrowserCommand
			browserGOOS = tt.goos
			var gotName string
			var gotArgs []string
			startBrowserCommand = func(name string, args ...string) error {
				gotName = name
				gotArgs = append([]string(nil), args...)
				return nil
			}
			t.Cleanup(func() {
				browserGOOS = originalGOOS
				startBrowserCommand = originalStart
			})

			require.NoError(t, platformOpenBrowserURL("https://example.com/reviews"))
			assert.Equal(t, tt.wantName, gotName)
			assert.Equal(t, tt.wantArgs, gotArgs)
			assert.NotEqual(t, "sh", gotName)
			assert.NotEqual(t, "cmd", gotName)
		})
	}
}
