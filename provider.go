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

type Provider struct {
	URL *url.URL
}

func NewProvider(urlStr string) (*Provider, error) {
	u, err := url.Parse(urlStr)
	if err != nil {
		return nil, err
	}

	return &Provider{
		URL: u,
	}, nil
}

type JSONRPCResponse struct {
	Version string          `json:"jsonrpc"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *JSONRPCError   `json:"error,omitempty"`
	ID      int             `json:"id"`
}

type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type JSONRPCBatchResponse []JSONRPCResponse

type JSONRPCRequest struct {
	ID     int             `json:"id"`
	Method string          `json:"method"`
	Params json.RawMessage `json:"params"`
}

type JSONRPCBatchRequest []JSONRPCRequest

// Check if request is a batch JSON-RPC request or not
func isBatchJSONRPCRequest(req *http.Request) (bool, error) {
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return false, err
	}

	req.Body = io.NopCloser(bytes.NewBuffer(body))

	var batchReq JSONRPCBatchRequest
	err = json.Unmarshal(body, &batchReq)

	if err != nil {
		// Error occurred. Treat as not a batch request.
		return false, nil
	}

	// It is a batch JSON-RPC request.
	return true, nil
}

// Send JSON-RPC request and handle errors in response
func (p *Provider) SendRequest(req *http.Request) (*http.Response, error) {
	isBatch, err := isBatchJSONRPCRequest(req)
	if err != nil {
		return nil, err
	}

	newReq, err := http.NewRequest(req.Method, p.URL.String(), req.Body)
	if err != nil {
		return nil, err
	}

	newReq.Header = req.Header

	resp, err := http.DefaultClient.Do(newReq)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		var buf bytes.Buffer
		_, err := buf.ReadFrom(resp.Body)
		if err != nil {
			return nil, err
		}
		return nil, errors.New(buf.String())
	}

	if resp.Header.Get("Content-Encoding") == "gzip" {
		gzipReader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return nil, err
		}
		defer gzipReader.Close()

		resp.Header.Del("Content-Encoding")

		body, err := io.ReadAll(gzipReader)
		if err != nil {
			return nil, err
		}

		resp.Body = io.NopCloser(bytes.NewReader(body))
	}

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}

	resp.Body.Close()

	if isBatch {
		var batchJsonResp JSONRPCBatchResponse
		err = json.Unmarshal(body, &batchJsonResp)
		if err != nil {
			return nil, err
		}

		for _, resp := range batchJsonResp {
			if resp.Error != nil {
				return nil, errors.New(resp.Error.Message)
			}
		}
	} else {
		var jsonResp JSONRPCResponse
		err = json.Unmarshal(body, &jsonResp)
		if err != nil {
			return nil, err
		}

		if jsonResp.Error != nil {
			return nil, errors.New(jsonResp.Error.Message)
		}
	}

	newResp := http.Response{
		Status:        resp.Status,
		StatusCode:    resp.StatusCode,
		Proto:         resp.Proto,
		ProtoMajor:    resp.ProtoMajor,
		ProtoMinor:    resp.ProtoMinor,
		ContentLength: int64(len(body)),
		Header:        resp.Header,
		Body:          io.NopCloser(bytes.NewReader(body)),
	}

	return &newResp, nil
}
