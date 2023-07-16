package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
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

		// for simplicity, just set content type to JSON
		w.Header().Set("Content-Type", "application/json")

		body, _ := io.ReadAll(resp.Body)
		w.Write(body)
	})

	fmt.Println("Starting server on port 8080")
	log.Fatal().Err(http.ListenAndServe(":8080", nil))
}

// ForwardRequest is a method that takes a JSON-RPC request and forwards it to the appropriate backend.
func (p *Proxy) ForwardRequest(req *http.Request) (*http.Response, error) {
	// Parse the JSON-RPC request
	var rpcRequest interface{}
	err := json.NewDecoder(req.Body).Decode(&rpcRequest)
	if err != nil {
		return nil, err
	}

	// Check if the request is a batch request
	isBatch := false
	var batchRequests []interface{}
	switch rpcRequest := rpcRequest.(type) {
	case []interface{}:
		batchRequests = rpcRequest
		isBatch = true
	case map[string]interface{}:
		batchRequests = []interface{}{rpcRequest}
		isBatch = false
	default:
		return nil, errors.New("invalid JSON-RPC request format")
	}

	// Create a response slice to hold the responses for each request
	responses := make([]*http.Response, len(batchRequests))

	for i, batchReq := range batchRequests {
		// Extract the method from the individual request
		method, ok := extractMethod(batchReq)
		if !ok {
			responses[i] = createErrorResponse(http.StatusBadRequest, "Method not found in request")
			continue
		}

		// Get the balancer for the method
		balancer, ok := p.BalancerMapping[method]
		if !ok {
			// If there isn't a specific balancer, use the general one
			balancer = p.Balancer
		}

		// Create a new http.Request for the individual request
		individualReq := cloneRequest(req)
		setMethodInRequest(individualReq, batchReq.(map[string]interface{}))

		// Forward the individual request with failover logic
		response, err := p.FailoverRequest(individualReq, balancer)
		if err != nil {
			responses[i] = createErrorResponse(http.StatusInternalServerError, err.Error())
			continue
		}

		responses[i] = response
	}

	// Create the response based on the request format
	var httpResponse *http.Response
	if isBatch {
		// Create a batch response
		batchResponse := createBatchResponse(responses)

		// Marshal the batch response into JSON
		responseBytes, err := json.Marshal(batchResponse)
		if err != nil {
			return nil, err
		}

		// Create a new http.Response with the batch response
		httpResponse = &http.Response{
			StatusCode: http.StatusOK,
			Header:     make(http.Header),
			Body:       io.NopCloser(bytes.NewReader(responseBytes)),
		}
	} else {
		// Use the response of the single request
		httpResponse = responses[0]
	}

	return httpResponse, nil
}

// Helper function to clone a http.Request
func cloneRequest(r *http.Request) *http.Request {
	clone := r.Clone(r.Context())
	clone.Header = make(http.Header)
	for k, vs := range r.Header {
		clone.Header[k] = vs
	}
	return clone
}

// Helper function to set method in http.Request
func setMethodInRequest(r *http.Request, req map[string]interface{}) {
	req["jsonrpc"] = "2.0"
	reqBytes, _ := json.Marshal(req)
	r.Body = io.NopCloser(bytes.NewReader(reqBytes))
	r.ContentLength = int64(len(reqBytes))
}

// Helper function to create an error response
func createErrorResponse(statusCode int, errMsg string) *http.Response {
	errResponse := map[string]interface{}{
		"jsonrpc": "2.0",
		"error": map[string]interface{}{
			"code":    statusCode,
			"message": errMsg,
		},
		"id": nil,
	}

	errResponseBytes, _ := json.Marshal(errResponse)

	return &http.Response{
		StatusCode: statusCode,
		Header:     make(http.Header),
		Body:       io.NopCloser(bytes.NewReader(errResponseBytes)),
	}
}

// Helper function to create a batch response
func createBatchResponse(responses []*http.Response) []interface{} {
	batchResponse := make([]interface{}, len(responses))
	for i, response := range responses {
		if response != nil {
			// Read the response body
			responseBody, _ := io.ReadAll(response.Body)
			response.Body.Close()

			// Parse the response body JSON
			var responseMap map[string]interface{}
			_ = json.Unmarshal(responseBody, &responseMap)

			// Set the individual response in the batch response
			batchResponse[i] = responseMap
		}
	}
	return batchResponse
}

// Helper function to extract the method from the JSON-RPC request
func extractMethod(request interface{}) (string, bool) {
	requestMap, ok := request.(map[string]interface{})
	if !ok {
		return "", false
	}

	method, ok := requestMap["method"].(string)
	if !ok {
		return "", false
	}

	return method, true
}
