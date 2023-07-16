package main

import (
	"bytes"
	"errors"
	"io"
	"net/http"
	"net/url"
)

// Provider is a struct that holds the URL of the backend service it should forward requests to.
type Provider struct {
	URL *url.URL
}

// NewProvider is a constructor function that creates a new Provider.
func NewProvider(urlStr string) (*Provider, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}
	return &Provider{
		URL: u,
	}, nil
}

// type jsonResponse struct {
// 	// Define fields for JSON-RPC response attributes
// 	Error *struct {
// 		Code    int    `json:"code"`
// 		Message string `json:"message"`
// 	} `json:"error"`
// 	Result interface{} `json:"result"`
// }

// SendRequest is a method that takes an http.Request, updates the URL, and sends it to the backend service.
// It returns an http.Response and an error if there was a problem sending the request or receiving the response.
func (p *Provider) SendRequest(req *http.Request) (*http.Response, error) {
	// Create a new HTTP request
	newReq, err := http.NewRequest(req.Method, p.URL.String(), req.Body)
	if err != nil {
		return nil, err
	}

	// Copy headers
	newReq.Header = req.Header

	// Send the new request
	resp, err := http.DefaultClient.Do(newReq)
	if err != nil {
		return nil, err
	}

	// Check the response for error conditions
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("non-OK response returned")
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Close the response body to prevent resource leaks
	resp.Body.Close()

	// Create a new response with the same values as the original response
	newResp := http.Response{
		Status:        resp.Status,
		StatusCode:    resp.StatusCode,
		Proto:         resp.Proto,
		ProtoMajor:    resp.ProtoMajor,
		ProtoMinor:    resp.ProtoMinor,
		ContentLength: int64(len(body)),
		Header:        resp.Header,
		Body:          io.NopCloser(bytes.NewReader(body)), // Create a new ReadCloser from the body
	}

	// Return the new response
	return &newResp, nil
}
