package replicator

import (
	"context"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestNewRequester(t *testing.T) {
	r := newRequester(requesterConfiguration{
		timeout:      10 * time.Second,
		userName:     "admin",
		userPassword: "admin",
	})

	assert.NotNil(t, r)

	assert.NotNil(t, r.client)
	assert.Equal(t, 10*time.Second, r.client.Timeout)

	assert.NotEmpty(t, r.b64Authorization)
	uEnc := b64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("admin:admin")))
	assert.Equal(t, uEnc, r.b64Authorization)
}

const (
	testAuthString  = "testAuthString"
	testBodyContent = "TestContent"
)

type testCase int

const (
	validResponse testCase = iota
	emptyBody
	wrongContentLength
	emptyB64String
	returnJsonBodyFromReq
)

func createMockServer(t *testing.T, tc testCase) *httptest.Server {
	server := httptest.NewServer(http.HandlerFunc(func(writer http.ResponseWriter, request *http.Request) {
		if tc == emptyB64String {
			assert.Empty(t, request.Header.Get("Authorization"))
			return
		}

		authHeader := request.Header.Get("Authorization")
		assert.NotEmpty(t, authHeader)
		assert.Contains(t, authHeader, "Basic")
		assert.Contains(t, authHeader, testAuthString)

		switch tc {
		case validResponse:
			writer.WriteHeader(http.StatusOK)
			writer.Write([]byte(testBodyContent))
		case emptyBody:
			writer.WriteHeader(http.StatusOK)
			writer.Write(nil)
		case wrongContentLength:
			writer.Header().Set("Content-Length", "1")
		case returnJsonBodyFromReq:
			writer.WriteHeader(http.StatusOK)
			requestBody, err := io.ReadAll(request.Body)
			require.NoError(t, err)
			writer.Write(requestBody)
		default:
			panic("unhandled default case")
		}
	}))

	return server
}

func TestDefaultRequester_Do(t *testing.T) {
	t.Run("Valid request", func(t *testing.T) {
		testServer := createMockServer(t, validResponse)
		defer testServer.Close()

		r := defaultRequester{
			client:           testServer.Client(),
			b64Authorization: testAuthString,
		}

		req, err := http.NewRequest(http.MethodGet, testServer.URL, nil)
		require.NoError(t, err)

		resp, err := r.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.NotNil(t, resp.ByteBody)
		assert.Equal(t, []byte(testBodyContent), resp.ByteBody)
	})

	t.Run("Empty b64AuthString", func(t *testing.T) {
		testServer := createMockServer(t, emptyB64String)
		defer testServer.Close()

		r := defaultRequester{
			client:           testServer.Client(),
			b64Authorization: "",
		}

		req, err := http.NewRequest(http.MethodGet, testServer.URL, nil)
		require.NoError(t, err)

		resp, err := r.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("Empty response body", func(t *testing.T) {
		testServer := createMockServer(t, emptyBody)
		defer testServer.Close()

		r := defaultRequester{
			client:           testServer.Client(),
			b64Authorization: testAuthString,
		}

		req, err := http.NewRequest(http.MethodGet, testServer.URL, nil)
		require.NoError(t, err)

		resp, err := r.Do(req)
		assert.NoError(t, err)
		assert.NotNil(t, resp)

		assert.Equal(t, http.StatusOK, resp.StatusCode)
		assert.NotNil(t, resp.ByteBody)
		assert.Empty(t, resp.ByteBody)
	})

	t.Run("Wrong content length in header", func(t *testing.T) {
		testServer := createMockServer(t, wrongContentLength)
		defer testServer.Close()

		r := defaultRequester{
			client:           testServer.Client(),
			b64Authorization: testAuthString,
		}

		req, err := http.NewRequest(http.MethodGet, testServer.URL, nil)
		require.NoError(t, err)

		resp, err := r.Do(req)
		assert.Nil(t, resp)
		assert.Error(t, err)
	})

	t.Run("req with exceeded deadline", func(t *testing.T) {
		testServer := createMockServer(t, wrongContentLength)
		defer testServer.Close()

		r := defaultRequester{
			client:           testServer.Client(),
			b64Authorization: testAuthString,
		}

		deadCtx, cancel := context.WithTimeout(context.TODO(), 0*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(deadCtx, http.MethodGet, testServer.URL, nil)
		require.NoError(t, err)

		resp, err := r.Do(req)
		assert.Nil(t, resp)
		assert.Error(t, err)
	})

}

func TestDefaultRequester_Send(t *testing.T) {
	t.Run("Valid request", func(t *testing.T) {
		testServer := createMockServer(t, validResponse)
		defer testServer.Close()

		r := defaultRequester{
			client:           testServer.Client(),
			b64Authorization: testAuthString,
		}

		resp, err := r.Send(http.MethodGet, testServer.URL)
		assert.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("Invalid method", func(t *testing.T) {
		testServer := createMockServer(t, validResponse)
		defer testServer.Close()

		r := defaultRequester{
			client:           testServer.Client(),
			b64Authorization: testAuthString,
		}

		resp, err := r.Send("\"", testServer.URL)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}

func TestDefaultRequester_SendWithJsonBody(t *testing.T) {
	t.Run("Valid request", func(t *testing.T) {
		testServer := createMockServer(t, returnJsonBodyFromReq)
		defer testServer.Close()

		r := defaultRequester{
			client:           testServer.Client(),
			b64Authorization: testAuthString,
		}

		type TestJson struct {
			Name string `json:"name"`
		}

		tj := TestJson{Name: "test"}

		resp, err := r.SendWithJsonBody(http.MethodPost, testServer.URL, tj)
		assert.NoError(t, err)
		assert.NotNil(t, resp)

		var tjCopy TestJson
		err = json.Unmarshal(resp.ByteBody, &tjCopy)
		assert.NoError(t, err)
		assert.Equal(t, tj, tjCopy)
	})

	t.Run("Invalid json", func(t *testing.T) {
		testServer := createMockServer(t, returnJsonBodyFromReq)
		defer testServer.Close()

		r := defaultRequester{
			client:           testServer.Client(),
			b64Authorization: testAuthString,
		}

		resp, err := r.SendWithJsonBody(http.MethodPost, testServer.URL, func() {})
		assert.Error(t, err)
		assert.Nil(t, resp)
	})

	t.Run("Invalid method in request", func(t *testing.T) {
		testServer := createMockServer(t, returnJsonBodyFromReq)
		defer testServer.Close()

		r := defaultRequester{
			client:           testServer.Client(),
			b64Authorization: testAuthString,
		}

		type TestJson struct {
			Name string `json:"name"`
		}

		tj := TestJson{Name: "test"}

		resp, err := r.SendWithJsonBody("\"", testServer.URL, tj)
		assert.Error(t, err)
		assert.Nil(t, resp)
	})
}
