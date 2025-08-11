package authorization

import (
	"github.com/stretchr/testify/assert"
	"net/http"
	"testing"
)

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
