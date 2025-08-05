package proxy

import (
	"bytes"
	"io"
	"net/http"
	"net/http/httptest"
	"net/http/httputil"
	"net/url"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/vulcand/oxy/v2/forward"
)

const helloWorldJson = `{"hello":"world"}`

type authHandler struct {
	originalServerAddress string
	proxySrv              *httputil.ReverseProxy
}

func (ah *authHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	targetURL, err := url.Parse(ah.originalServerAddress)
	if err != nil {
		panic("could not parse target url " + ah.originalServerAddress + " " + err.Error())
	}
	r.URL.Host = targetURL.Host
	r.URL.Scheme = targetURL.Scheme
	ah.proxySrv.ServeHTTP(w, r)
}

func TestCheckBody(t *testing.T) {
	testSrv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		body, err := io.ReadAll(r.Body)
		require.NoError(t, err)
		assert.Equal(t, body, []byte(helloWorldJson))

		w.WriteHeader(200)
		_, err = w.Write([]byte("hello golang" + string(body)))
		require.NoError(t, err)
	}))
	defer testSrv.Close()

	req, err := http.NewRequest(http.MethodPost, testSrv.URL+"/", bytes.NewBuffer([]byte(helloWorldJson)))
	req.Header["Content-Type"] = []string{"application/x-www-form-urlencoded"}
	require.NoError(t, err)

	proxySrv := forward.New(true)

	authMiddleWare := &authHandler{
		originalServerAddress: testSrv.URL,
		proxySrv:              proxySrv,
	}

	http.NewServeMux().Handle("/", authMiddleWare)

	do, err := http.DefaultClient.Do(req)
	require.NoError(t, err)
	defer do.Body.Close()

	assert.Equal(t, http.StatusOK, do.StatusCode)
	actualRespBody := []byte{}
	_, err = do.Body.Read(actualRespBody)
	require.NoError(t, err)
	assert.Equal(t, "asdf", actualRespBody)
}

func TestLeTest(t *testing.T) {
	bodyNopCloser := io.NopCloser(bytes.NewReader([]byte(helloWorldJson)))
	limRead := io.LimitReader(bodyNopCloser, 17)
	bodyBytes := make([]byte, 17)

	_, err := limRead.Read(bodyBytes)
	require.NoError(t, err)

	assert.Len(t, bodyBytes, 17)
	assert.Equal(t, []byte(helloWorldJson), bodyBytes)
}
