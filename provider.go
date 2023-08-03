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
	ID      int64           `json:"id"`
}

type JSONRPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

type JSONRPCBatchResponse []JSONRPCResponse

func (p *Provider) SendRequest(req *http.Request) (*http.Response, error) {
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

	if len(body) == 0 {
		return nil, errors.New("received empty response")
	}

	var responses JSONRPCBatchResponse
	if err = json.Unmarshal(body, &responses); err != nil {
		var singleResponse JSONRPCResponse
		if err = json.Unmarshal(body, &singleResponse); err != nil {
			return nil, err
		}
		if singleResponse.Error != nil && singleResponse.Error.Message != "execution reverted" {
			return nil, errors.New(singleResponse.Error.Message)
		}
	} else {
		for _, response := range responses {
			if response.Error != nil && response.Error.Message != "execution reverted" {
				return nil, errors.New(response.Error.Message)
			}
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
