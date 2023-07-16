package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"net/http"

	"github.com/rs/zerolog/log"
	"golang.org/x/time/rate"
)

// Proxy is a struct that holds the map of JSON-RPC methods to backend URLs and a slice of Provider clients.
type Proxy struct {
	BalancerMapping map[string]*Balancer
	Balancer        *Balancer
	RetryLimit      int
	Limiter         *rate.Limiter
}

// NewProxy is a constructor function that creates a new Proxy.
func NewProxy(config Config) *Proxy {
	// Create a slice of all Provider clients
	allProviders := make([]*Provider, len(config.Providers))
	var err error
	for i, providerURL := range config.Providers {
		allProviders[i], err = NewProvider(providerURL)
		if err != nil {
			log.Warn().Msgf("Error creating provider: %v", err)
			continue
		}
	}

	// Create a new Balancer for all providers
	balancer := &Balancer{
		Providers: allProviders,
	}

	// Create a new rate limiter
	limiter := rate.NewLimiter(rate.Limit(config.RequestsPerSecond), config.Burst)

	// Create a mapping from methods to Balancers
	balancerMapping := make(map[string]*Balancer)

	for _, mapping := range config.MethodsMapping {
		method := mapping.Method
		providers := make([]*Provider, len(mapping.Providers))
		for i, url := range mapping.Providers {
			for _, provider := range allProviders {
				if provider.URL.String() == url {
					providers[i] = provider
					break
				}
			}
		}
		balancerMapping[method] = &Balancer{Providers: providers}
	}

	// Create a new Proxy
	return &Proxy{
		BalancerMapping: balancerMapping,
		Balancer:        balancer,
		RetryLimit:      config.RetryLimit,
		Limiter:         limiter,
	}
}

// Start is a method that starts the proxy.
func (p *Proxy) Start() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		resp, err := p.ForwardRequest(r)
		if err != nil {
			log.Err(err).Msg("Error forwarding request")
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		// Copy the headers from the original response to the response writer's headers
		for key, values := range resp.Header {
			for _, value := range values {
				w.Header().Add(key, value)
			}
		}

		body, _ := ioutil.ReadAll(resp.Body)
		w.Write(body)
	})

	fmt.Println("Starting server on port 8080")
	log.Fatal().Err(http.ListenAndServe(":8080", nil))
}

// ForwardRequest is a method that takes a JSON-RPC request and forwards it to the appropriate backend.
func (p *Proxy) ForwardRequest(req *http.Request) (*http.Response, error) {
	// Parse the JSON-RPC method from the request
	method, err := parseMethod(req)
	if err != nil {
		return nil, err
	}

	// Get the balancer for the method
	balancer, ok := p.BalancerMapping[method]
	if !ok {
		// If there isn't a specific balancer, use the general one
		balancer = p.Balancer
	}

	// Use the failover logic with the specific balancer
	return p.FailoverRequest(req, balancer)
}

// parseMethod is a helper function that parses the JSON-RPC method from the request.
func parseMethod(req *http.Request) (string, error) {
	// Check if the request body is nil
	if req.Body == nil {
		return "", errors.New("request body is nil")
	}

	// Read the request body into a byte slice
	bodyBytes, err := ioutil.ReadAll(req.Body)
	if err != nil {
		return "", err
	}

	// Create a new ReadCloser with the byte slice
	req.Body = ioutil.NopCloser(bytes.NewReader(bodyBytes))

	// Decode the request body into a map
	var payload map[string]interface{}
	err = json.Unmarshal(bodyBytes, &payload)
	if err != nil {
		return "", err
	}

	// Extract the method from the payload
	method, ok := payload["method"].(string)
	if !ok {
		return "", errors.New("method not found in payload")
	}

	return method, nil
}
