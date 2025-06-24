package authentication

import (
	"net/http"
	"strings"

	"github.com/cloudogu/sonarcarp/internal"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("sonarcarp")

type IsAuthenticated func(w http.ResponseWriter, r *http.Request) bool
type RedirectToLogin func(w http.ResponseWriter, r *http.Request)
type RedirectToLogout func(w http.ResponseWriter, r *http.Request)
type IsFirstAuthenticatedRequest func(r *http.Request) bool
type Username func(r *http.Request) string
type Attributes func(r *http.Request) map[string][]string

type Authenticator struct {
	IsAuthenticated
	RedirectToLogin
	RedirectToLogout
	IsFirstAuthenticatedRequest
	Username
	Attributes
	ForwardUnauthenticatedRESTRequests bool
}

type middlewareHandler interface {
	Handle(h http.Handler) http.Handler
}

type CasClientSet struct {
	BrowserClient middlewareHandler
	RestClient    middlewareHandler
}

type MiddlewareConfiguration struct {
	CasClientSet
	Authenticator
}

func CreateMiddleware(cfg MiddlewareConfiguration) func(http.Handler) http.Handler {
	browserClient := cfg.BrowserClient
	restClient := cfg.RestClient

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			log.Debugf("AuthenticationMiddleware called with request %+v", r.Header)

			// casHandler initializes the general cas flow by setting the cas client in the request and checking for
			// back channel logouts
			casHandler := browserClient.Handle

			// authenticationHandler handles the actual authentication
			authenticationHandler := authenticationMiddleware(next, cfg.Authenticator)

			restHandler := casHandler(restClient.Handle(authenticationHandler))
			browserHandler := casHandler(authenticationHandler)

			if isBrowserRequest(r) {
				log.Debugf("Request is browser request")
				browserHandler.ServeHTTP(w, r)

				return
			}

			log.Debugf("Request is REST request")

			if cfg.ForwardUnauthenticatedRESTRequests {
				log.Debugf("Unauthenticated REST request is allowed")
				next.ServeHTTP(w, r.WithContext(WithUnauthenticatedRequest(r.Context())))

				return
			}

			restHandler.ServeHTTP(w, r)
		})
	}
}

func authenticationMiddleware(next http.Handler, a Authenticator) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !a.IsAuthenticated(w, r) {
			log.Debugf("Unauthenticated user - redirect to login")
			a.RedirectToLogin(w, r)

			return
		}

		if strings.HasSuffix(r.URL.String(), "/logout") {
			log.Debugf("received logout request")
			a.RedirectToLogout(w, r)

			return
		}

		user := internal.User{
			UserName:   a.Username(r),
			Replicate:  a.IsFirstAuthenticatedRequest(r),
			Attributes: a.Attributes(r),
		}

		log.Debugf("Request by user %v", user)

		userCtx := internal.WithUser(r.Context(), user)
		userReq := r.WithContext(userCtx)

		next.ServeHTTP(w, userReq)
	})
}

func isBrowserRequest(req *http.Request) bool {
	userAgent := req.Header.Get("User-Agent")

	return strings.Contains(strings.ToLower(userAgent), "mozilla")
}

func GetCasIsAuthenticated(isAuthenticated func(r *http.Request) bool) IsAuthenticated {
	return func(w http.ResponseWriter, r *http.Request) bool {
		authenticated := isAuthenticated(r)

		if authenticated && r.URL.Query().Has("ticket") && isBrowserRequest(r) {
			log.Debugf("Removing service ticket from request by redirecting...")
			query := r.URL.Query()
			delete(query, "ticket")
			r.URL.RawQuery = query.Encode()
			http.Redirect(w, r, r.URL.String(), http.StatusTemporaryRedirect)
		}

		return authenticated
	}
}
