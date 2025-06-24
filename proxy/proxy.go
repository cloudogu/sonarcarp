package proxy

import (
	"fmt"
	"net/http"
	"net/url"

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
}

func (p proxyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !p.authorizationChecker.IsAuthorized(r) {
		log.Infof("Found unauthorized request: IP %s, X-RealIP %s, URL %s", r.RemoteAddr, r.Header[forward.XRealIP], r.URL.String())
		p.unauthorizedServer.ServeUnauthorized(w, r)

		return
	}

	log.Debug("Found authorized request: IP %s, XForwardedFor %s, URL %s", r.RemoteAddr, r.Header[forward.XForwardedFor], r.URL.String())
	r.URL.Host = p.targetURL.Host     // copy target URL but not the URL path, only the host
	r.URL.Scheme = p.targetURL.Scheme // (and scheme because they get lost on the way)
	p.forwarder.ServeHTTP(w, r)
}
func applyMiddleware(h http.Handler, middlewares ...middleware) http.Handler {
	for i := len(middlewares) - 1; i >= 0; i-- {
		h = middlewares[i](h)
	}
	return h
}
func createProxyHandler(sTargetURL string, us unauthorizedServer, ac authorizationChecker, middlewares ...middleware) (http.Handler, error) {
	targetURL, err := url.Parse(sTargetURL)
	if err != nil {
		return proxyHandler{}, fmt.Errorf("could not parse target url '%s': %w", sTargetURL, err)
	}

	fwd := forward.New(true)

	pHandler := proxyHandler{
		targetURL:            targetURL,
		forwarder:            fwd,
		unauthorizedServer:   us,
		authorizationChecker: ac,
	}

	return applyMiddleware(pHandler, middlewares...), nil
}
