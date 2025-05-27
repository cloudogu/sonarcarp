package proxy

import (
	"fmt"
	"github.com/vulcand/oxy/forward"
	"net/http"
	"net/url"
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
		p.unauthorizedServer.ServeUnauthorized(w, r)

		return
	}

	r.URL = p.targetURL
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

	fwd, err := forward.New(forward.PassHostHeader(true))
	if err != nil {
		return proxyHandler{}, fmt.Errorf("failed to create forward %w", err)
	}

	pHandler := proxyHandler{
		targetURL:            targetURL,
		forwarder:            fwd,
		unauthorizedServer:   us,
		authorizationChecker: ac,
	}

	return applyMiddleware(pHandler, middlewares...), nil
}
