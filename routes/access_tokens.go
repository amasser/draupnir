package routes

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/gocardless/draupnir/routes/chain"
	"github.com/google/jsonapi"
	"github.com/pkg/errors"
	"golang.org/x/net/context"
	"golang.org/x/oauth2"
)

const TOKEN_EXCHANGE_TIMEOUT = time.Second * 5
const OAUTH_CALLBACK_TIMEOUT = time.Minute

type AccessTokens struct {
	Callbacks map[string]chan OAuthCallback
	Client    OAuthClient
}

type OAuthCallback struct {
	Token oauth2.Token
	Error error
}

// OAuthClient is the abstract interface for handling OAuth.
// Both the real OAuth client and our fake for testing
// will implement this interface
type OAuthClient interface {
	AuthCodeURL(string, ...oauth2.AuthCodeOption) string
	Exchange(context.Context, string) (*oauth2.Token, error)
}

func (a AccessTokens) Authenticate(w http.ResponseWriter, r *http.Request) error {
	r.ParseForm()
	state := r.Form.Get("state")

	url := a.Client.AuthCodeURL(state, oauth2.AccessTypeOffline)

	w.Header().Add("Location", url)
	w.WriteHeader(http.StatusFound)
	return nil
}

type createAccessTokenRequest struct {
	State string `jsonapi:"attr,state"`
}

// Create completes the OAuth flow and returns an access token
//
// The flow for this is a bit tricky, so it's worth going through.
// When we receive a request to create an access token, we create a channel and
// store it in the Callbacks map, keyed by the state parameter provided in the
// request. We then block on the channel, waiting to receive an OAuthCallback
// through it.
// The client will send the user through the OAuth flow, providing the same
// state parameter. When the user finishes the flow, they'll be redirected to
// the Callback handler, which is also in this route set.
// The Callback handler will handle the redirect, exchanging the authorisation
// code for an access token if it was successful, and will send the outcome
// through the same channel (looking it up by the state).
// Create will then receive the result through the channel, remove the channel
// from the map, and serialise a result back to the client.
func (a AccessTokens) Create(w http.ResponseWriter, r *http.Request) error {
	var req createAccessTokenRequest

	logger, err := GetLogger(r)
	if err != nil {
		return err
	}

	if err := jsonapi.UnmarshalPayload(r.Body, &req); err != nil {
		RenderError(w, http.StatusBadRequest, invalidJSONError)
		return nil
	}

	state := req.State

	callback := make(chan OAuthCallback)
	a.Callbacks[state] = callback

	token, err := waitForCallback(callback)
	delete(a.Callbacks, state)

	if err != nil {
		logger.With("error", err.Error()).Info("oauth request failed")
		RenderError(w, http.StatusBadRequest, oauthError) // TODO: improve error
		return nil
	}

	w.WriteHeader(http.StatusCreated)
	err = json.NewEncoder(w).Encode(token)
	if err != nil {
		return errors.Wrap(err, "failed to encode access token")
	}
	return nil
}

func waitForCallback(callbackChan chan OAuthCallback) (oauth2.Token, error) {
	select {
	case c := <-callbackChan:
		if c.Error != nil {
			return oauth2.Token{}, c.Error
		}
		return c.Token, nil
	case <-time.After(OAUTH_CALLBACK_TIMEOUT):
		return oauth2.Token{}, errors.New("Callback timed out")
	}
}

func (a AccessTokens) Callback(w http.ResponseWriter, r *http.Request) error {
	logger, err := GetLogger(r)
	if err != nil {
		return err
	}

	r.ParseForm()

	respError := r.Form.Get("error")
	respCode := r.Form.Get("code")
	state := r.Form.Get("state")

	callback := a.Callbacks[state]
	if callback == nil {
		logger.With("state", state).Info("cannot find oauth callback for state")
		return nil
	}

	if respError != "" {
		err := errors.New(respError)
		callback <- OAuthCallback{Error: err}
		return err
	}

	if respCode == "" {
		err := fmt.Errorf("OAuth callback response code is empty")
		callback <- OAuthCallback{Error: err}
		// TODO: remove this and log the state earlier?
		logger.With("state", state).Error("empty oauth response code")
		return err
	}

	ctx, cancel := context.WithTimeout(r.Context(), TOKEN_EXCHANGE_TIMEOUT)
	defer cancel()
	token, err := a.Client.Exchange(ctx, respCode)

	if err != nil {
		err := errors.Wrap(err, "token exchange error")
		callback <- OAuthCallback{Error: err}
		return err
	}

	callback <- OAuthCallback{Token: *token}

	w.Header().Set("Content-Type", "text/html")
	w.Write([]byte("<h1>Success!</h1><h3>You can close this tab</h3><script>window.close()</script>"))
	return nil
}

func OauthErrorRenderer(next chain.Handler) chain.Handler {
	return func(w http.ResponseWriter, r *http.Request) error {
		err := next(w, r)
		if err != nil {
			w.Header().Set("Content-Type", "text/html")
			w.WriteHeader(http.StatusInternalServerError)
			response := fmt.Sprintf(
				`<h1>Error</h1>
				 <h3>There was an error. Please try again</h3>
				 <pre>%s</pre>`,
				err.Error(),
			)
			w.Write([]byte(response))
		}
		return err
	}
}
