package proxy

import (
	"fmt"
	"net/http"
	"strconv"

	"github.com/cloudogu/carp"
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

	casClientFactory, err := carp.NewCasClientFactory(carp.Configuration{
		CasUrl:                             configuration.CasUrl,
		ServiceUrl:                         configuration.ServiceUrl,
		SkipSSLVerification:                configuration.SkipSSLVerification,
		ForwardUnauthenticatedRESTRequests: configuration.ForwardUnauthenticatedRESTRequests,
	})
	if err != nil {
		return nil, fmt.Errorf("failed to create cas client factory: %w", err)
	}

	casClient := casClientFactory.CreateClient()

	headers := authorization.Headers{
		Principal: configuration.PrincipalHeader,
		Role:      configuration.RoleHeader,
		Mail:      configuration.MailHeader,
		Name:      configuration.NameHeader,
	}

	router := http.NewServeMux()

	pHandler, err := createProxyHandler(
		configuration.Target,
		headers,
		casClient,
	)

	router.Handle("/", pHandler)

	if len(configuration.CarpResourcePath) != 0 {
		router.Handle(configuration.CarpResourcePath, http.StripPrefix(configuration.CarpResourcePath, loggingMiddleware(staticResourceHandler)))
	}

	log.Debugf("starting server on port %d", configuration.Port)

	return &http.Server{
		Addr:    ":" + strconv.Itoa(configuration.Port),
		Handler: router,
	}, nil
}
