package replicator

import (
	"errors"
	"github.com/cloudogu/sonarcarp/internal"
	"github.com/op/go-logging"
	"net/http"
	"time"
)

var log = logging.MustGetLogger("sonarcarp")

var ErrNotFound = errors.New("NotFound")

type groupModifier interface {
	addMissingGroups(userId userId, casGroups []string) error
	removeNonExistingGroups(userId userId, casGroups []string) error
}

type userModifier interface {
	createOrGetUser(user internal.User) (userId, error)
}

type requester interface {
	Send(method, url string) (*ResponseWithBody, error)
	SendWithJsonBody(method, url string, jsonStruct any) (*ResponseWithBody, error)
}

type DefaultReplicator struct {
	userModifier  userModifier
	groupModifier groupModifier
}

type Endpoints struct {
	UserEndpoints
	GroupEndpoints
}

type Configuration struct {
	RequestUserName     string
	RequestUserPassword string
	Endpoints
}

func NewReplicator(configuration Configuration) *DefaultReplicator {
	r := newRequester(requesterConfiguration{
		timeout:      10 * time.Second,
		userName:     configuration.RequestUserName,
		userPassword: configuration.RequestUserPassword,
	})

	return &DefaultReplicator{
		userModifier: defaultUserModifier{
			UserEndpoints: configuration.Endpoints.UserEndpoints,
			requester:     r,
		},
		groupModifier: defaultGroupModifier{
			GroupEndpoints: configuration.Endpoints.GroupEndpoints,
			requester:      r,
		},
	}
}

func CreateMiddleware(cfg Configuration) func(http.Handler) http.Handler {
	replicator := NewReplicator(cfg)

	return createMiddleware(replicator)
}

func createMiddleware(r *DefaultReplicator) func(next http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
			log.Debugf("replicator middleware called")

			ctxUser, ok := internal.GetUser(request.Context())
			if !ok {
				next.ServeHTTP(writer, request)

				return
			}

			log.Debugf("found user in context %+v", ctxUser)

			if !ctxUser.Replicate {
				log.Debugf("user doesn't need to be replicated")
				next.ServeHTTP(writer, request)

				return
			}

			r.Replicate(ctxUser)

			log.Debugf("User has been replicated")

			next.ServeHTTP(writer, request)
		})
	}
}

func (u *DefaultReplicator) Replicate(user internal.User) {
	log.Debugf("Try to replicator groups for user %s (%T)", user.UserName, user.Attributes)
	log.Debugf("Found groups: %v", user.Attributes["groups"])

	uId, err := u.userModifier.createOrGetUser(user)
	if err != nil {
		log.Warningf("error while creating or updating user: %v", err)
		return
	}

	err = u.groupModifier.addMissingGroups(uId, user.GetGroups())
	if err != nil {
		log.Warningf("error while adding missing groups: %v", err)
	}

	err = u.groupModifier.removeNonExistingGroups(uId, user.GetGroups())
	if err != nil {
		log.Warningf("error while removing non existing groups: %v", err)
	}
}
