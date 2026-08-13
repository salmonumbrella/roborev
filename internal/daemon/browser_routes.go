package daemon

import (
	"context"
	"time"

	"github.com/danielgtaylor/huma/v2"
)

type WebLoginRequest struct {
	Token string `json:"token" minLength:"1"`
}

type WebSessionCredentials struct {
	Session   string    `json:"session"`
	CSRF      string    `json:"csrf"`
	ExpiresAt time.Time `json:"expires_at"`
}

type WebSessionStatus struct {
	Authentication string     `json:"authentication" enum:"local,token"`
	Authenticated  bool       `json:"authenticated"`
	ExpiresAt      *time.Time `json:"expires_at,omitempty"`
}

type WebLoginInput struct {
	Body WebLoginRequest
}

type WebBootstrapInput struct {
	Body struct{}
}

type WebSessionCredentialsOutput struct {
	Body WebSessionCredentials
}

type WebSessionStatusOutput struct {
	Body WebSessionStatus
}

type WebLogoutOutput struct{}

func (s *Server) registerBrowserRoutes(api huma.API) {
	huma.Post(api, "/api/ui/session/login", unavailableWebLogin,
		func(operation *huma.Operation) {
			operation.OperationID = "login-web-session"
			operation.Summary = "Exchange a daemon token for a browser session"
			operation.Tags = []string{"web-session"}
		})
	huma.Post(api, "/api/ui/session/bootstrap", unavailableWebBootstrap,
		func(operation *huma.Operation) {
			operation.OperationID = "bootstrap-web-session"
			operation.Summary = "Mint tab credentials from an ambient browser session"
			operation.Tags = []string{"web-session"}
		})
	huma.Delete(api, "/api/ui/session", unavailableWebLogout,
		func(operation *huma.Operation) {
			operation.OperationID = "logout-web-session"
			operation.Summary = "Invalidate a browser session"
			operation.Tags = []string{"web-session"}
			operation.DefaultStatus = 204
		})
	huma.Get(api, "/api/ui/session", unavailableWebSessionStatus,
		func(operation *huma.Operation) {
			operation.OperationID = "get-web-session-status"
			operation.Summary = "Get browser authentication status"
			operation.Tags = []string{"web-session"}
		})
}

func unavailableWebLogin(context.Context, *WebLoginInput) (*WebSessionCredentialsOutput, error) {
	return nil, huma.Error404NotFound("browser session routes require the browser listener")
}

func unavailableWebBootstrap(context.Context, *WebBootstrapInput) (*WebSessionCredentialsOutput, error) {
	return nil, huma.Error404NotFound("browser session routes require the browser listener")
}

func unavailableWebLogout(context.Context, *struct{}) (*WebLogoutOutput, error) {
	return nil, huma.Error404NotFound("browser session routes require the browser listener")
}

func unavailableWebSessionStatus(context.Context, *struct{}) (*WebSessionStatusOutput, error) {
	return nil, huma.Error404NotFound("browser session routes require the browser listener")
}
