package main

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"

	"github.com/rs/zerolog/log"
)

// FailoverRequest tries to send the request to each provider in the balancer.
// If all providers fail, it returns an error.
func (p *Proxy) FailoverRequest(req *http.Request, balancer *Balancer, method string) (*http.Response, error) {
	var lastError error
	retries := 0

	// Store the body
	body, err := io.ReadAll(req.Body)
	if err != nil {
		return nil, err
	}
	_ = req.Body.Close()

	for {
		// Wait for the rate limiter to allow another request
		p.Limiter.Wait(context.Background())

		// Get next provider and loop back to the start if we've reached the end
		provider := balancer.NextProvider()

		// Create a new ReadCloser for each request
		clonedReq := req.Clone(context.Background())
		clonedReq.Body = io.NopCloser(bytes.NewBuffer(body))
		resp, err := provider.SendRequest(clonedReq)
		if err == nil {
			log.Debug().Msgf("✅ [%s] %s", method, provider.URL.String())
			return resp, nil
		}

		log.Err(err).Msgf("❌ Error sending request to provider %s", provider.URL.String())
		lastError = err
		retries++

		// Check retry limit
		if p.RetryLimit != -1 && retries >= p.RetryLimit {
			break
		}

		// Reset provider list if we've tried all providers
		if retries%len(balancer.Providers) == 0 {
			balancer.ResetProviderList()
		}
	}

	return nil, errors.New("All providers failed: " + lastError.Error())
}
