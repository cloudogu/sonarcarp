package authorization

import (
	"context"
	"github.com/cloudogu/sonarcarp/authentication"
	"github.com/cloudogu/sonarcarp/internal"
	"github.com/cloudogu/sonarcarp/mocks"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
	"net/http"
	"net/http/httptest"
	"testing"
)

func TestGroupsWithAccess(t *testing.T) {
	tests := []struct {
		name          string
		groups        []string
		authenticated bool
		expectAccess  bool
		expectedRole  string
		userNotOk     bool
	}{
		{name: "no group has no access", groups: []string{}, expectAccess: false},
		{name: "random group has no access", groups: []string{"randomgrp"}, expectAccess: false},
		{name: "admin group has access", groups: []string{"admin"}, expectAccess: true, expectedRole: "Admin"},
		{name: "gadmin group has access", groups: []string{"gadmin"}, expectAccess: true, expectedRole: "Admin"},
		{name: "writer group has access", groups: []string{"gwriter"}, expectAccess: true, expectedRole: "Editor"},
		{name: "reader group has access", groups: []string{"greader"}, expectAccess: true, expectedRole: "Viewer"},
		{name: "unauthenticated requests are forwarded", groups: []string{}, authenticated: true},
		{name: "user does not exist", expectAccess: false, userNotOk: true},
		{name: "all groups at the same time: user has access", groups: []string{"admin", "gadmin", "reader", "writer"}, expectAccess: true, expectedRole: "Admin"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			configuration := MiddlewareConfiguration{
				Headers: Headers{
					Principal: "xprin",
					Role:      "xrole",
					Mail:      "xmail",
					Name:      "xname",
				},
				Groups: Groups{
					CesAdmin: "admin",
					Admin:    "gadmin",
					Reader:   "greader",
					Writer:   "gwriter",
				},
			}
			middleware := CreateMiddleware(configuration)
			mh := &mocks.Handler{
				MserveHTTP: func(w http.ResponseWriter, r *http.Request) {
					if test.expectAccess {
						assert.NotEmpty(t, r.Header.Get(configuration.Headers.Principal))
					} else {
						assert.Empty(t, r.Header.Get(configuration.Headers.Principal))
					}
				},
			}
			if !test.userNotOk {
				mh.On("ServeHTTP", mock.Anything, mock.Anything)
			}
			handler := middleware(mh)
			mw := httptest.NewRecorder()
			var ctx context.Context
			if test.authenticated {
				ctx = authentication.WithUnauthenticatedRequest(context.TODO())
			} else {
				ctx = context.TODO()
			}

			if !test.userNotOk {
				ctx = internal.WithUser(ctx, internal.User{
					UserName:  "testuser",
					Replicate: false,
					Attributes: map[string][]string{
						"groups": test.groups,
					},
				})
			}
			r, err := http.NewRequestWithContext(ctx, "", "", nil)
			require.NoError(t, err)

			handler.ServeHTTP(mw, r)

			if test.userNotOk {
				assert.Equal(t, mw.Code, http.StatusInternalServerError)
				assert.Equal(t, mw.Body.String(), "Could not extract user from request")
			}
			mh.AssertExpectations(t)
		})
	}
}

func TestIsAuthorized(t *testing.T) {
	authHeader = "testauthheader"
	t.Run("is  authorized", func(t *testing.T) {
		header := http.Header{}
		header.Set(authHeader, "1")
		isAuth := IsAuthorized(&http.Request{
			Header: header,
		})
		assert.True(t, isAuth)
	})
	t.Run("is not authorized", func(t *testing.T) {
		isAuth := IsAuthorized(&http.Request{})
		assert.False(t, isAuth)
	})
}
