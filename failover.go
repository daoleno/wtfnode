package main

import (
	"context"
	"errors"
	"net/http"

	"github.com/rs/zerolog/log"
)

// FailoverRequest tries to send the request to each provider in the balancer.
// If all providers fail, it returns an error.

func (p *Proxy) FailoverRequest(req *http.Request, balancer *Balancer, method string) (*http.Response, error) {
	var lastError error
	retries := 0

	for i := 0; i < len(balancer.Providers); i++ {
		// Wait for the rate limiter to allow another request
		p.Limiter.Wait(context.Background())
		provider := balancer.NextProvider()
		resp, err := provider.SendRequest(req)
		if err == nil {
			log.Debug().Msgf("✅ [%s] %s", method, provider.URL.String())
			return resp, nil
		}
		log.Err(err).Msgf("❌ Error sending request to provider %s", provider.URL.String())
		lastError = err

		retries++
		if p.RetryLimit != -1 && retries >= p.RetryLimit {
			break
		}
	}

	return nil, errors.New("All providers failed: " + lastError.Error())
}
