package proxy

import (
	"fmt"
	"net/http"
	"net/http/httputil"
	"net/url"

	"github.com/cloudogu/go-cas"
	"github.com/cloudogu/sonarcarp/authorization"
	"github.com/vulcand/oxy/v2/forward"
)

type middleware func(http.Handler) http.Handler

type authorizationChecker interface {
	IsAuthorized(r *http.Request) bool
}

type unauthorizedServer interface {
	ServeUnauthorized(writer http.ResponseWriter, req *http.Request)
}

type proxyHandler struct {
	targetURL            *url.URL
	forwarder            http.Handler
	unauthorizedServer   unauthorizedServer
	authorizationChecker authorizationChecker
	casClient            *cas.Client
	headers              authorization.Headers
}

func createProxyHandler(sTargetURL string, headers authorization.Headers, casClient *cas.Client) (http.Handler, error) {
	log.Debugf("creating proxy middleware")

	targetURL, err := url.Parse(sTargetURL)
	if err != nil {
		return proxyHandler{}, fmt.Errorf("could not parse target url '%s': %w", sTargetURL, err)
	}

	fwd := forward.New(true)

	return doEverythingEverywhereAllAtOnce(fwd, casClient, targetURL, headers), nil
}

func doEverythingEverywhereAllAtOnce(fwd *httputil.ReverseProxy, casClient *cas.Client, targetURL *url.URL, headers authorization.Headers) http.Handler {
	pHandler := proxyHandler{
		targetURL: targetURL,
		forwarder: fwd,
		casClient: casClient,
		headers:   headers,
	}

	return casClient.Handle(pHandler)
}

func (p proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !cas.IsAuthenticated(r) {
		cas.RedirectToLogin(w, r)
		return
	}

	if r.URL.Path == "/logout" {
		cas.RedirectToLogout(w, r)
		return
	}

	log.Debugf("proxy middleware called with request to %s and headers %+v", r.URL.String(), r.Header)

	log.Debug("Found authorized request: IP %s, XForwardedFor %s, URL %s", r.RemoteAddr, r.Header[forward.XForwardedFor], r.URL.String())
	r.URL.Host = p.targetURL.Host     // copy target URL but not the URL path, only the host
	r.URL.Scheme = p.targetURL.Scheme // (and scheme because they get lost on the way)

	setHeaders(r, p.headers)

	p.forwarder.ServeHTTP(w, r)
}

func setHeaders(r *http.Request, headers authorization.Headers) {
	r.Header.Add(headers.Principal, cas.Username(r))

	attrs := cas.Attributes(r)
	r.Header.Add(headers.Name, attrs.Get("displayName"))
	r.Header.Add(headers.Mail, attrs.Get("mail"))
	r.Header.Add(headers.Role, attrs.Get("groups"))
}
