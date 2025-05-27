package replicator

import (
	"bytes"
	b64 "encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

type ResponseWithBody struct {
	*http.Response
	ByteBody []byte
}

type requesterConfiguration struct {
	timeout      time.Duration
	userName     string
	userPassword string
}

func newRequester(cfg requesterConfiguration) *defaultRequester {
	return &defaultRequester{
		client:           &http.Client{Timeout: cfg.timeout},
		b64Authorization: b64.StdEncoding.EncodeToString([]byte(fmt.Sprintf("%s:%s", cfg.userName, cfg.userPassword))),
	}
}

type defaultRequester struct {
	client           *http.Client
	b64Authorization string
}

func (d defaultRequester) setAuthHeader(req *http.Request) {
	if len(d.b64Authorization) == 0 {
		return
	}

	req.Header.Set("Authorization", fmt.Sprintf("Basic %s", d.b64Authorization))

	log.Debugf("Added Authorization header to request.")
}

func (d defaultRequester) Do(req *http.Request) (*ResponseWithBody, error) {
	log.Debugf("Do %s request to url '%s'", req.Method, req.URL)

	d.setAuthHeader(req)

	resp, err := d.client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to do request: %w", err)
	}

	defer func() {
		_ = resp.Body.Close()
	}()

	responseBody, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, fmt.Errorf("failed to read response body: %w", err)
	}

	log.Debugf("Request to '%s' returned status code '%v'", req.URL, resp.StatusCode)
	log.Debugf("Response body: %s", responseBody)

	return &ResponseWithBody{
		Response: resp,
		ByteBody: responseBody,
	}, nil
}

func (d defaultRequester) SendWithJsonBody(method, url string, jsonStruct any) (*ResponseWithBody, error) {
	jsonBytes, err := json.Marshal(jsonStruct)
	if err != nil {
		return nil, fmt.Errorf("unable to marshal struct %v: %w", jsonStruct, err)
	}

	req, err := http.NewRequest(method, url, bytes.NewBuffer(jsonBytes))
	if err != nil {
		return nil, fmt.Errorf("could not create request for url %s: %w", url, err)
	}

	req.Header.Set("Content-Type", "application/json")

	return d.Do(req)
}

func (d defaultRequester) Send(method, url string) (*ResponseWithBody, error) {
	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, fmt.Errorf("could not create request for url %s: %w", url, err)
	}

	return d.Do(req)
}
