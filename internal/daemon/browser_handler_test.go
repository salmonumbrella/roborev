package daemon

import (
	"bytes"
	"crypto/rand"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func newBrowserHandlerFixture(t *testing.T, authToken string) (http.Handler, *BrowserSessionManager) {
	t.Helper()
	policy, err := NewBrowserPolicy(BrowserEndpoint{
		Address:           "127.0.0.1:7374",
		Origin:            "http://127.0.0.1:7374",
		Enabled:           true,
		remoteAuthEnabled: authToken != "",
	}, "")
	require.NoError(t, err)
	sessions, err := NewBrowserSessionManager(BrowserSessionConfig{
		Origin:     "http://127.0.0.1:7374",
		AuthToken:  authToken,
		AllowLocal: authToken == "",
		Entropy:    rand.Reader,
		Clock:      time.Now,
	})
	require.NoError(t, err)
	core := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"path":"` + request.URL.Path + `"}`))
	})
	static := http.HandlerFunc(func(w http.ResponseWriter, request *http.Request) {
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("shell"))
	})
	server := &Server{}
	handler, err := server.newBrowserHandler(core, static, policy, sessions)
	require.NoError(t, err)
	return handler, sessions
}

func browserRequest(method, path string, body any) *http.Request {
	var data []byte
	if body != nil {
		data, _ = json.Marshal(body)
	}
	request := httptest.NewRequest(method, "http://127.0.0.1:7374"+path, bytes.NewReader(data))
	request.Host = "127.0.0.1:7374"
	request.RemoteAddr = "127.0.0.1:50000"
	request.Header.Set("Origin", "http://127.0.0.1:7374")
	if body != nil {
		request.Header.Set("Content-Type", "application/json")
	}
	return request
}

func TestBrowserHandlerHostRunsBeforeAuthentication(t *testing.T) {
	handler, _ := newBrowserHandlerFixture(t, "secret")
	request := browserRequest(http.MethodGet, "/api/status", nil)
	request.Host = "attacker.test"
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusBadRequest, recorder.Code)
	assert.JSONEq(t, `{"error":"invalid_host"}`, recorder.Body.String())
}

func TestBrowserHandlerLoginBootstrapAndAuthenticatedRoutes(t *testing.T) {
	handler, sessions := newBrowserHandlerFixture(t, "secret")
	login := browserRequest(http.MethodPost, "/api/ui/session/login", WebLoginRequest{Token: "secret"})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, login)
	require.Equal(t, http.StatusOK, recorder.Code)
	var credentials WebSessionCredentials
	require.NoError(t, json.Unmarshal(recorder.Body.Bytes(), &credentials))
	assert.NotEmpty(t, credentials.Session)
	assert.NotEmpty(t, credentials.CSRF)
	cookies := recorder.Result().Cookies()
	require.Len(t, cookies, 1)
	assert.Equal(t, sessions.CookieName(), cookies[0].Name)

	status := browserRequest(http.MethodGet, "/api/status", nil)
	status.AddCookie(cookies[0])
	status.Header.Set(WebSessionHeader, credentials.Session)
	statusRecorder := httptest.NewRecorder()
	handler.ServeHTTP(statusRecorder, status)
	assert.Equal(t, http.StatusOK, statusRecorder.Code)
	assert.Equal(t, "private, no-store", statusRecorder.Header().Get("Cache-Control"))

	bootstrap := browserRequest(http.MethodPost, "/api/ui/session/bootstrap", map[string]any{})
	bootstrap.AddCookie(cookies[0])
	bootstrap.Header.Set("Sec-Fetch-Site", "same-origin")
	bootstrap.Header.Set("Sec-Fetch-Mode", "cors")
	bootstrap.Header.Set("Sec-Fetch-Dest", "empty")
	bootstrapRecorder := httptest.NewRecorder()
	handler.ServeHTTP(bootstrapRecorder, bootstrap)
	assert.Equal(t, http.StatusOK, bootstrapRecorder.Code)
	assert.Empty(t, bootstrapRecorder.Header().Values("Set-Cookie"))
}

func TestBrowserHandlerCSRFAllowlistAndPublicShell(t *testing.T) {
	handler, sessions := newBrowserHandlerFixture(t, "secret")
	credentials, err := sessions.Login("secret")
	require.NoError(t, err)

	mutation := browserRequest(http.MethodPost, "/api/job/cancel", map[string]int{"job_id": 7})
	mutation.AddCookie(sessions.Cookie(credentials.Ambient))
	mutation.Header.Set(WebSessionHeader, credentials.Tab)
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, mutation)
	assert.Equal(t, http.StatusForbidden, recorder.Code)

	mutation.Header.Set(WebCSRFHeader, credentials.CSRF)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, mutation)
	assert.Equal(t, http.StatusOK, recorder.Code)

	for _, path := range []string{"/api/shutdown", "/api/unknown", "/debug/pprof/", "/openapi.json"} {
		request := browserRequest(http.MethodGet, path, nil)
		recorder = httptest.NewRecorder()
		handler.ServeHTTP(recorder, request)
		assert.Equal(t, http.StatusNotFound, recorder.Code, path)
	}

	shell := browserRequest(http.MethodGet, "/reviews/7", nil)
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, shell)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.Equal(t, "shell", recorder.Body.String())
}

func TestBrowserHandlerLocalBootstrapRequiresFetchMetadata(t *testing.T) {
	handler, _ := newBrowserHandlerFixture(t, "")
	request := browserRequest(http.MethodPost, "/api/ui/session/bootstrap", map[string]any{})
	recorder := httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusForbidden, recorder.Code)

	request = browserRequest(http.MethodPost, "/api/ui/session/bootstrap", map[string]any{})
	request.Header.Set("Sec-Fetch-Site", "same-origin")
	request.Header.Set("Sec-Fetch-Mode", "cors")
	request.Header.Set("Sec-Fetch-Dest", "empty")
	recorder = httptest.NewRecorder()
	handler.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusOK, recorder.Code)
	assert.NotEmpty(t, recorder.Header().Get("Set-Cookie"))
}

func TestBrowserSessionRoutesAppearOnlyInCombinedOpenAPI(t *testing.T) {
	spec, err := OpenAPISpec()
	require.NoError(t, err)
	assert.Contains(t, string(spec), `"/api/ui/session/login"`)
	assert.Contains(t, string(spec), `"login-web-session"`)

	server := &Server{}
	mux := http.NewServeMux()
	server.registerHumaAPI(mux)
	request := httptest.NewRequest(http.MethodPost, "/api/ui/session/login", nil)
	recorder := httptest.NewRecorder()
	mux.ServeHTTP(recorder, request)
	assert.Equal(t, http.StatusNotFound, recorder.Code)
}
