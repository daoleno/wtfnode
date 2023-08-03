package main

import (
	"bytes"
	"compress/gzip"
	"encoding/json"
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

// Structure for JSON-RPC response
type JSONRPCResponse struct {
	Version string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      int             `json:"id"`
}

// Structure for JSON-RPC error
type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Send JSON-RPC request and handle errors in response
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
		var buf bytes.Buffer
		_, err := buf.ReadFrom(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(buf.String())
	}

	// Check if response is encoded with gzip
	if resp.Header.Get("Content-Encoding") == "gzip" {
		// Decode response body as gzip
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gzipReader.Close()

		// delete the Content-Encoding header
		resp.Header.Del("Content-Encoding")

		// Read the decompressed response body
		body, err := io.ReadAll(gzipReader)
		if err != nil {
			return nil, err
		}

		// Set the decompressed body as the new response body
		resp.Body = io.NopCloser(bytes.NewReader(body))
	}

	// Read the response body
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	// Close the response body to prevent resource leaks
	resp.Body.Close()

	// Parse the JSON-RPC response
	var jsonResp JSONRPCResponse
	err = json.Unmarshal(body, &jsonResp)
	if err != nil {
		return nil, err
	}

	// Check if an error exists
	if jsonResp.Error != nil {
		return nil, errors.New(jsonResp.Error.Message)
	}

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
