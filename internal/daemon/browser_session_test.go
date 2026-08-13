package daemon

import (
	"bytes"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newTestBrowserSessions(t *testing.T, allowLocal bool) (*BrowserSessionManager, *time.Time) {
	t.Helper()
	now := time.Date(2026, time.August, 12, 12, 0, 0, 0, time.UTC)
	manager, err := NewBrowserSessionManager(BrowserSessionConfig{
		Origin:     "https://reviews.example.com",
		AuthToken:  "configured-secret",
		AllowLocal: allowLocal,
		TTL:        time.Hour,
		Entropy:    bytes.NewReader(bytes.Repeat([]byte("entropy-for-browser-sessions-"), 1000)),
		Clock:      func() time.Time { return now },
	})
	require.NoError(t, err)
	return manager, &now
}

func TestBrowserSessionLoginBootstrapAndAuthentication(t *testing.T) {
	manager, _ := newTestBrowserSessions(t, true)
	for _, token := range []string{"", "wrong"} {
		_, err := manager.Login(token)
		require.ErrorIs(t, err, ErrInvalidWebToken)
	}

	first, err := manager.Login("configured-secret")
	require.NoError(t, err)
	assert.NotContains(t, first.Ambient+first.Tab+first.CSRF, "configured-secret")
	principal, err := manager.Authenticate(first.Ambient, first.Tab)
	require.NoError(t, err)
	assert.False(t, principal.Local)
	require.NoError(t, manager.CheckCSRF(first.Tab, first.CSRF))

	second, err := manager.Bootstrap(first.Ambient)
	require.NoError(t, err)
	assert.NotEqual(t, first.Tab, second.Tab)
	assert.NotEqual(t, first.CSRF, second.CSRF)
	_, err = manager.Authenticate(first.Ambient, first.Tab)
	require.NoError(t, err, "bootstrap must not invalidate the first tab")
	_, err = manager.Authenticate(first.Ambient, second.Tab)
	require.NoError(t, err)
	require.ErrorIs(t, manager.CheckCSRF(first.Tab, second.CSRF), ErrInvalidWebCSRF)
}

func TestBrowserSessionRejectsCrossSessionAndRestartCredentials(t *testing.T) {
	firstManager, _ := newTestBrowserSessions(t, true)
	secondManager, _ := newTestBrowserSessions(t, true)
	first, err := firstManager.Login("configured-secret")
	require.NoError(t, err)
	second, err := firstManager.Login("configured-secret")
	require.NoError(t, err)

	_, err = firstManager.Authenticate(first.Ambient, second.Tab)
	require.ErrorIs(t, err, ErrWebSessionRequired)
	_, err = secondManager.Authenticate(first.Ambient, first.Tab)
	require.ErrorIs(t, err, ErrWebSessionRequired)
}

func TestBrowserSessionExpiryLogoutAndLocalPolicy(t *testing.T) {
	manager, now := newTestBrowserSessions(t, true)
	credentials, err := manager.NewLocalSession()
	require.NoError(t, err)
	principal, err := manager.Authenticate(credentials.Ambient, credentials.Tab)
	require.NoError(t, err)
	assert.True(t, principal.Local)

	manager.Logout(credentials.Ambient)
	_, err = manager.Authenticate(credentials.Ambient, credentials.Tab)
	require.ErrorIs(t, err, ErrWebSessionRequired)

	expiring, err := manager.Login("configured-secret")
	require.NoError(t, err)
	*now = now.Add(2 * time.Hour)
	_, err = manager.Authenticate(expiring.Ambient, expiring.Tab)
	require.ErrorIs(t, err, ErrWebSessionRequired)

	manager, _ = newTestBrowserSessions(t, false)
	_, err = manager.NewLocalSession()
	require.ErrorIs(t, err, ErrLocalWebSessionDisabled)
}

func TestBrowserSessionCookieScope(t *testing.T) {
	manager, _ := newTestBrowserSessions(t, true)
	cookie := manager.Cookie("ambient-value")
	assert.True(t, strings.HasPrefix(cookie.Name, "roborev_web_"))
	assert.Equal(t, "ambient-value", cookie.Value)
	assert.Empty(t, cookie.Domain)
	assert.Equal(t, "/", cookie.Path)
	assert.True(t, cookie.HttpOnly)
	assert.True(t, cookie.Secure)
	assert.Equal(t, http.SameSiteStrictMode, cookie.SameSite)

	expired := manager.ExpiredCookie()
	assert.Equal(t, cookie.Name, expired.Name)
	assert.Equal(t, -1, expired.MaxAge)
	assert.True(t, expired.Expires.Before(time.Unix(1, 0)))
}

func TestBrowserSessionConcurrentBootstrapAuthenticateAndLogout(t *testing.T) {
	manager, _ := newTestBrowserSessions(t, true)
	credentials, err := manager.Login("configured-secret")
	require.NoError(t, err)

	var wait sync.WaitGroup
	for range 20 {
		wait.Go(func() {
			tab, bootstrapErr := manager.Bootstrap(credentials.Ambient)
			if bootstrapErr == nil {
				_, _ = manager.Authenticate(credentials.Ambient, tab.Tab)
			}
		})
	}
	wait.Go(func() {
		manager.Logout(credentials.Ambient)
	})
	wait.Wait()
}
