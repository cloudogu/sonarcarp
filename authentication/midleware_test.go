package authentication

import (
	"context"
	"fmt"
	"github.com/cloudogu/sonarcarp/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"net/url"
	"testing"
)

func TestCreateMiddleware(t *testing.T) {
	tests := []struct {
		name             string
		isBrowserRequest bool
		forwardRest      bool
	}{
		{name: "forward browser request", isBrowserRequest: true},
		{name: "forward rest", isBrowserRequest: false, forwardRest: true},
		{name: "do normal forward", isBrowserRequest: false, forwardRest: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mh := &mocks.Handler{}
			mw := httptest.NewRecorder()
			mwh := &mockMiddlewareHandler{}
			mwh.On("Handle", mock.Anything).Return(mh)
			mh.On("ServeHTTP", mock.Anything)
			configuration := MiddlewareConfiguration{
				CasClientSet: CasClientSet{
					BrowserClient: mwh,
					RestClient:    mwh,
				},
				Authenticator: Authenticator{
					IsAuthenticated:                    func(w http.ResponseWriter, r *http.Request) bool { return false },
					RedirectToLogin:                    func(w http.ResponseWriter, r *http.Request) {},
					RedirectToLogout:                   func(w http.ResponseWriter, r *http.Request) {},
					IsFirstAuthenticatedRequest:        func(r *http.Request) bool { return false },
					Username:                           func(r *http.Request) string { return "" },
					Attributes:                         func(r *http.Request) map[string][]string { return map[string][]string{} },
					ForwardUnauthenticatedRESTRequests: test.forwardRest,
				},
			}

			middleware := CreateMiddleware(configuration)
			handler := middleware(mh)
			ctx := context.TODO()
			r, err := http.NewRequestWithContext(ctx, "", "https://test.de?ticket=a", nil)
			require.NoError(t, err)

			if test.isBrowserRequest {
				header := http.Header{}
				header.Set("User-Agent", "mozilla")
				r.Header = header
			}

			require.NoError(t, err)

			handler.ServeHTTP(mw, r)

			mh.AssertExpectations(t)
			mwh.AssertExpectations(t)
		})
	}
}

func TestAuthenticationMiddleware(t *testing.T) {
	tests := []struct {
		name          string
		authenticated bool
		isLogout      bool
	}{
		{name: "unauthenticated is redirected to login", authenticated: false},
		{name: "logout request is redirected", authenticated: true, isLogout: true},
		{name: "authenticated request ist forwarded", authenticated: true, isLogout: false},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			mh := &mocks.Handler{}
			mw := httptest.NewRecorder()
			RedirectToLogoutCalls := 0
			authenticator := Authenticator{
				IsAuthenticated: func(w http.ResponseWriter, r *http.Request) bool { return test.authenticated },
				RedirectToLogin: func(w http.ResponseWriter, r *http.Request) {
				},
				RedirectToLogout: func(w http.ResponseWriter, r *http.Request) {
					RedirectToLogoutCalls++
				},
				IsFirstAuthenticatedRequest: func(r *http.Request) bool { return false },
				Username:                    func(r *http.Request) string { return "" },
				Attributes:                  func(r *http.Request) map[string][]string { return map[string][]string{} },
			}
			ctx := context.TODO()
			r, err := http.NewRequestWithContext(ctx, "", "", nil)
			require.NoError(t, err)
			if test.isLogout {
				r.URL, err = url.Parse("https://test.de/logout")
				require.NoError(t, err)
			}

			if !test.isLogout && test.authenticated {
				mh.On("ServeHTTP", mock.Anything, mock.Anything)
			}

			middleware := authenticationMiddleware(mh, authenticator)
			middleware.(http.HandlerFunc)(mw, r)

			if test.isLogout {
				assert.Equal(t, RedirectToLogoutCalls, 1)
			}

			mh.AssertExpectations(t)
		})
	}
}

func TestCasIsAuthenticated(t *testing.T) {
	tests := []struct {
		name             string
		authenticated    bool
		isBrowserRequest bool
		hasTicket        bool
	}{
		{name: "redirect is not called (authenticated=true, browser-request=true, service-ticket=no)", authenticated: true, isBrowserRequest: true, hasTicket: false},
		{name: "redirect is called (authenticated=true, browser-request=true, service-ticket=yes)", authenticated: true, isBrowserRequest: true, hasTicket: true},
		{name: "redirect is not called (authenticated=true, browser-request=false, service-ticket=no)", authenticated: true, isBrowserRequest: false, hasTicket: false},
		{name: "redirect is not called (authenticated=true, browser-request=false, service-ticket=yes)", authenticated: true, isBrowserRequest: false, hasTicket: true},
		{name: "redirect is not called (authenticated=false, browser-request=true, service-ticket=no)", authenticated: false, isBrowserRequest: true, hasTicket: false},
		{name: "redirect is not called (authenticated=false, browser-request=true, service-ticket=yes)", authenticated: false, isBrowserRequest: true, hasTicket: true},
		{name: "redirect is not called (authenticated=false, browser-request=false, service-ticket=no)", authenticated: false, isBrowserRequest: false, hasTicket: false},
		{name: "redirect is not called (authenticated=false, browser-request=false, service-ticket=yes)", authenticated: false, isBrowserRequest: false, hasTicket: true},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			isAuthenticated := func(r *http.Request) bool {
				return test.authenticated
			}
			var ticket string
			if test.hasTicket {
				ticket = "?ticket=a"
			}
			u, _ := url.Parse(fmt.Sprintf("https://www.test.de%s", ticket))
			header := map[string][]string{}
			if test.isBrowserRequest {
				header["User-Agent"] = []string{"mozilla"}
			}
			r := &http.Request{
				URL:    u,
				Header: header,
			}
			w := httptest.NewRecorder()

			result := GetCasIsAuthenticated(isAuthenticated)(w, r)

			assert.Equal(t, test.authenticated, result)

			if test.authenticated && test.hasTicket && test.isBrowserRequest {
				assert.Equal(t, http.StatusTemporaryRedirect, w.Code)
			} else {
				assert.Equal(t, http.StatusOK, w.Code)
			}
		})
	}
}
