package cashelper

import (
	"bytes"
	"github.com/cloudogu/go-cas"
	"io"
	"net/http"
)

type preserveBodyHandler struct {
	body            io.ReadCloser
	originalHandler http.Handler
}

func (b preserveBodyHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	r.Body = b.body
	b.originalHandler.ServeHTTP(w, r)
}

type mainHandler struct {
	originalHandler http.Handler
	casClient       *cas.Client
}

func (t mainHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	var bd []byte
	var err error

	if r.Body != nil {
		bd, err = io.ReadAll(r.Body)
		if err != nil {
			panic(err.Error())
		}
	}

	bh := preserveBodyHandler{
		body:            io.NopCloser(bytes.NewBuffer(bd)),
		originalHandler: t.originalHandler,
	}

	t.casClient.Handle(bh).ServeHTTP(w, r)
}

func NewHandler(originalHandler http.Handler, casClient *cas.Client) http.Handler {
	return mainHandler{
		originalHandler: originalHandler,
		casClient:       casClient,
	}
}
