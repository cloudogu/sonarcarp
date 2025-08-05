package authorization

import (
	"net/http"
	"strings"

	"github.com/cloudogu/sonarcarp/authentication"
	"github.com/cloudogu/sonarcarp/internal"
	"github.com/op/go-logging"
)

var (
	log        = logging.MustGetLogger("sonarcarp")
	authHeader string
)

type user struct {
	internal.User
}

type Headers struct {
	Principal string
	Role      string
	Mail      string
	Name      string
}

type Groups struct {
	CesAdmin string
}

type MiddlewareConfiguration struct {
	Headers Headers
}

func CreateMiddleware(config MiddlewareConfiguration) func(http.Handler) http.Handler {
	authHeader = config.Headers.Principal

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, r *http.Request) {
			log.Debugf("Authorization middleware called with request to %s and headers %+v", r.URL.String(), r.Header)

			ctx := r.Context()

			if authentication.UnauthenticatedRequestAllowed(ctx) {
				next.ServeHTTP(writer, r)
				return
			}

			ctxUser, ok := internal.GetUser(r.Context())
			if !ok {
				writer.WriteHeader(http.StatusInternalServerError)
				_, err := writer.Write([]byte("Could not extract user from request"))
				if err != nil {
					log.Warningf("Could not write error to response body: %v", err)
				}

				return
			}

			authUser := user{User: ctxUser}

			uGroups := authUser.GetGroups()
			log.Debugf("User %s has groups: %v", authUser.UserName, uGroups)

			setHeaders(r, authUser, config.Headers)
			log.Debugf("Set new request headers after authorization: %v", r.Header)

			next.ServeHTTP(writer, r)
		})
	}
}

func setHeaders(r *http.Request, u user, headers Headers) {
	r.Header.Set(headers.Principal, u.UserName)
	r.Header.Set(headers.Role, strings.Join(u.GetGroups(), ","))
	r.Header.Set(headers.Mail, u.GetMail())
	r.Header.Set(headers.Name, u.GetDisplayName())
}

type CheckerFunc func(r *http.Request) bool

func (c CheckerFunc) IsAuthorized(r *http.Request) bool {
	return c(r)
}

func IsAuthorized(r *http.Request) bool {
	if len(r.Header.Get(authHeader)) == 0 {
		return false
	}

	return true
}
