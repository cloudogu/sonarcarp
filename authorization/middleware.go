package authorization

import (
	"github.com/cloudogu/sonarcarp/authentication"
	"github.com/cloudogu/sonarcarp/internal"
	"github.com/op/go-logging"
	"net/http"
	"slices"
)

var (
	log        = logging.MustGetLogger("sonarcarp")
	authHeader string
)

type user struct {
	internal.User
	internal.Role
}

type Headers struct {
	Principal string
	Role      string
	Mail      string
	Name      string
}

type Groups struct {
	CesAdmin string
	Admin    string
	Reader   string
	Writer   string
}

type MiddlewareConfiguration struct {
	Headers Headers
	Groups  Groups
}

func CreateMiddleware(config MiddlewareConfiguration) func(http.Handler) http.Handler {
	authHeader = config.Headers.Principal
	groups := config.Groups

	allowedGroups := []string{
		groups.CesAdmin,
		groups.Admin,
		groups.Reader,
		groups.Writer,
	}

	log.Debugf("Allowed groups: '%v'", allowedGroups)

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			log.Debugf("Authorization middleware is called")

			ctx := request.Context()

			if authentication.UnauthenticatedRequestAllowed(ctx) {
				next.ServeHTTP(writer, request)
				return
			}

			ctxUser, ok := internal.GetUser(request.Context())
			if !ok {
				writer.WriteHeader(http.StatusInternalServerError)
				_, err := writer.Write([]byte("Could not extract user from request"))
				if err != nil {
					log.Warningf("Cloud not write error to response body: %v", err)
				}

				return
			}

			authUser := user{
				User: ctxUser,
				Role: ctxUser.GetRole(groups.CesAdmin, groups.Admin, groups.Writer),
			}

			uGroups := authUser.GetGroups()
			log.Debugf("User %s has groups: %v", authUser.UserName, uGroups)

			accessGranted := isAccessGranted(uGroups, allowedGroups)
			log.Debugf("Access granted for user '%s': %v", authUser.UserName, accessGranted)

			setHeaders(request, accessGranted, authUser, config.Headers)
			log.Debugf("Set new request headers after authorization: %v", request.Header)

			next.ServeHTTP(writer, request)
		})
	}
}

func isAccessGranted(userGroups []string, allowedGroups []string) bool {
	for _, group := range userGroups {
		if slices.Contains(allowedGroups, group) {
			return true
		}
	}

	return false
}

func setHeaders(r *http.Request, accessGranted bool, u user, headers Headers) {
	if !accessGranted {
		r.Header.Del(headers.Principal)

		return
	}

	r.Header.Set(headers.Principal, u.UserName)
	r.Header.Set(headers.Role, string(u.Role))
	r.Header.Set(headers.Mail, u.GetMail())
	r.Header.Set(headers.Name, u.GetMail())
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
