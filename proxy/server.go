package proxy

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/cloudogu/carp"
	"github.com/cloudogu/go-cas"
	"github.com/cloudogu/sonarcarp/authentication"
	"github.com/cloudogu/sonarcarp/authorization"
	"github.com/cloudogu/sonarcarp/config"
	"github.com/op/go-logging"
)

var log = logging.MustGetLogger("sonarcarp")

func NewServer(configuration config.Configuration) (*http.Server, error) {
	staticResourceHandler, err := createStaticFileHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to create static handler: %w", err)
	}

	clientFactory, err := carp.NewCasClientFactory(carp.Configuration{
		CasUrl:                             configuration.CasUrl,
		ServiceUrl:                         configuration.ServiceUrl,
		SkipSSLVerification:                configuration.SkipSSLVerification,
		ForwardUnauthenticatedRESTRequests: configuration.ForwardUnauthenticatedRESTRequests,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cas client factory: %w", err)
	}

	authenticationMiddleware := authentication.CreateMiddleware(authentication.MiddlewareConfiguration{
		CasClientSet: authentication.CasClientSet{
			BrowserClient: clientFactory.CreateClient(),
			RestClient:    clientFactory.CreateRestClient(),
		},
		Authenticator: authentication.Authenticator{
			ForwardUnauthenticatedRESTRequests: configuration.ForwardUnauthenticatedRESTRequests,
			IsAuthenticated:                    authentication.GetCasIsAuthenticated(cas.IsAuthenticated),
			RedirectToLogin:                    cas.RedirectToLogin,
			RedirectToLogout:                   cas.RedirectToLogout,
			IsFirstAuthenticatedRequest:        cas.IsFirstAuthenticatedRequest,
			Username:                           cas.Username,
			Attributes: func(r *http.Request) map[string][]string {
				return cas.Attributes(r)
			},
		},
	})

	authorizationMiddleware := authorization.CreateMiddleware(authorization.MiddlewareConfiguration{
		Headers: authorization.Headers{
			Principal: configuration.PrincipalHeader,
			Role:      configuration.RoleHeader,
			Mail:      configuration.MailHeader,
			Name:      configuration.NameHeader,
		},
	})

	router := http.NewServeMux()

	aChecker := authorization.CheckerFunc(authorization.IsAuthorized)
	pHandler, err := createProxyHandler(
		configuration.Target,
		staticResourceHandler,
		aChecker,
		// loggingMiddleware,
		authenticationMiddleware,
		authorizationMiddleware,
	)

	router.Handle("/", pHandler)

	if len(configuration.CarpResourcePath) != 0 {
		router.Handle(configuration.CarpResourcePath, http.StripPrefix(configuration.CarpResourcePath, loggingMiddleware(staticResourceHandler)))
	}

	return &http.Server{
		Addr:    ":" + strconv.Itoa(configuration.Port),
		Handler: router,
	}, nil
}
