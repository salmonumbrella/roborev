package daemon

import (
	"crypto/rand"
	"errors"
	"fmt"
	"log"
	"net/http"
	"time"

	"go.kenn.io/roborev/internal/config"
	webassets "go.kenn.io/roborev/internal/web"
)

var browserCapabilities = []string{"web-ui-v1", "web-session-v1"}

func (s *Server) startBrowserServer(web config.WebConfig) (*BrowserRuntimeInfo, error) {
	endpoint, err := ResolveBrowserEndpoint(web)
	if err != nil {
		return nil, err
	}
	if !endpoint.Enabled {
		return nil, nil
	}
	fail := func(err error) (*BrowserRuntimeInfo, error) {
		_ = endpoint.Listener.Close()
		return nil, err
	}
	policy, err := NewBrowserPolicy(endpoint, s.webDevOrigin)
	if err != nil {
		return fail(err)
	}
	sessionOrigin := endpoint.Origin
	if s.webDevOrigin != "" {
		sessionOrigin = s.webDevOrigin
	}
	sessions, err := NewBrowserSessionManager(BrowserSessionConfig{
		Origin:     sessionOrigin,
		AuthToken:  web.AuthToken,
		AllowLocal: web.AuthToken == "",
		Entropy:    rand.Reader,
		Clock:      time.Now,
	})
	if err != nil {
		return fail(err)
	}
	static, err := webassets.NewEmbeddedHandler()
	if err != nil {
		return fail(fmt.Errorf("load embedded browser application: %w", err))
	}
	handler, err := s.newBrowserHandler(s.httpServer.Handler, static, policy, sessions)
	if err != nil {
		return fail(err)
	}
	server := &http.Server{Addr: endpoint.Address, Handler: handler}
	s.endpointMu.Lock()
	s.browserServer = server
	s.browserListener = endpoint.Listener
	s.endpointMu.Unlock()
	serveErrCh := make(chan error, 1)
	go func() {
		serveErrCh <- server.Serve(endpoint.Listener)
	}()
	if err := waitForBrowserReady(endpoint, serveErrCh); err != nil {
		_ = server.Close()
		return nil, err
	}
	go func() {
		if serveErr := <-serveErrCh; serveErr != nil && !errors.Is(serveErr, http.ErrServerClosed) {
			log.Printf("Browser HTTP server stopped: %v", serveErr)
		}
	}()
	origin := endpoint.Origin
	if s.webDevOrigin != "" {
		origin = s.webDevOrigin
	}
	return &BrowserRuntimeInfo{
		Address:      endpoint.DialAddress,
		Origin:       origin,
		Capabilities: append([]string(nil), browserCapabilities...),
	}, nil
}

func waitForBrowserReady(endpoint BrowserEndpoint, serveErrCh <-chan error) error {
	client := &http.Client{Timeout: 200 * time.Millisecond}
	deadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(deadline) {
		select {
		case err := <-serveErrCh:
			if err == nil {
				return fmt.Errorf("browser server exited before ready")
			}
			return err
		default:
		}
		request, err := http.NewRequest(http.MethodGet, "http://"+endpoint.DialAddress+"/api/ping", nil)
		if err != nil {
			return err
		}
		request.Host = endpoint.Address
		response, err := client.Do(request)
		if err == nil {
			_ = response.Body.Close()
			if response.StatusCode == http.StatusOK {
				return nil
			}
		}
		time.Sleep(25 * time.Millisecond)
	}
	return fmt.Errorf("browser server did not become ready")
}
