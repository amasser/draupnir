package middleware

import (
	"context"
	"errors"
	"net/http"

	"github.com/gocardless/draupnir/server/api"
	"github.com/gocardless/draupnir/server/api/auth"
	"github.com/gocardless/draupnir/server/api/chain"
)

// This, sadly is exported so we can inject fake loggers in tests.
// See routes.createRequest in server/api/routes/fakes.go
const AuthUserKey key = 2

// Authenticate uses the provided authenticator to authenticate the request.
// On success, it yields to the next handler in the chain.
// On failure, it renders 401 Unauthorized.
func Authenticate(authenticator auth.Authenticator) chain.Middleware {
	return func(next chain.Handler) chain.Handler {
		return func(w http.ResponseWriter, r *http.Request) error {
			logger, err := GetLogger(r)
			if err != nil {
				return err
			}

			email, err := authenticator.AuthenticateRequest(r)
			if err != nil {
				logger.Info(err.Error())
				api.RenderError(w, http.StatusUnauthorized, api.UnauthorizedError)
				return nil
			}

			r = r.WithContext(context.WithValue(r.Context(), AuthUserKey, email))
			return next(w, r)
		}
	}
}

func GetAuthenticatedUser(r *http.Request) (string, error) {
	user, ok := r.Context().Value(AuthUserKey).(string)
	if !ok {
		return "", errors.New("Could not acquire authenticated user")
	}
	return user, nil
}
