package proxy

import (
	"crypto/tls"
	"fmt"
	"net/http"
	"net/url"
	"path"
	"strconv"

	"github.com/cloudogu/go-cas"
	"github.com/cloudogu/sonarcarp/config"
	"github.com/op/go-logging"
	"github.com/pkg/errors"
)

var log = logging.MustGetLogger("sonarcarp")

func NewServer(configuration config.Configuration) (*http.Server, error) {
	staticResourceHandler, err := createStaticFileHandler()
	if err != nil {
		return nil, fmt.Errorf("failed to create static handler: %w", err)
	}

	casClient, err := NewCasClientFactory(configuration)
	if err != nil {
		return nil, fmt.Errorf("failed to create CAS client: %w", err)
	}

	headers := authorizationHeaders{
		Principal: configuration.PrincipalHeader,
		Role:      configuration.RoleHeader,
		Mail:      configuration.MailHeader,
		Name:      configuration.NameHeader,
	}

	router := http.NewServeMux()

	pHandler, err := createProxyHandler(
		configuration.ServiceUrl,
		headers,
		casClient,
		configuration.LogoutPath,
		configuration.LogoutRedirectPath,
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

func NewCasClientFactory(configuration config.Configuration) (*cas.Client, error) {
	casUrl, err := url.Parse(configuration.CasUrl)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse cas url: %s", configuration.CasUrl)
	}

	serviceUrl, err := url.Parse(configuration.ServiceUrl)
	if err != nil {
		return nil, errors.Wrapf(err, "failed to parse service url: %s", configuration.ServiceUrl)
	}

	urlScheme := cas.NewDefaultURLScheme(casUrl)
	urlScheme.ServiceValidatePath = path.Join("p3", "serviceValidate")

	httpClient := &http.Client{}
	if configuration.SkipSSLVerification {
		transport := &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: true},
		}
		httpClient.Transport = transport
	}

	return cas.NewClient(&cas.Options{
		URL:       serviceUrl,
		Client:    httpClient,
		URLScheme: urlScheme,
		// Explicit disable the normal logout flow of go-cas as it consumes the form-body
		IsLogoutRequest: func(r *http.Request) bool {
			return false
		},
	}), nil
}
