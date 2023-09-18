package proxy

import (
	"sync"
)

// Balancer struct holds slice of Provider clients and a counter for round robin algorithm
type Balancer struct {
	Providers []*Provider
	Counter   int
	Mu        sync.Mutex
}

// NextProvider method returns the next provider to use based on the round robin load balancing algorithm
func (b *Balancer) NextProvider() *Provider {
	b.Mu.Lock()
	defer b.Mu.Unlock()

	if b.Counter >= len(b.Providers) {
		b.Counter = 0
	}

	selectedProvider := b.Providers[b.Counter]
	b.Counter++

	return selectedProvider
}

// ResetProviderList resets the counter in the Balancer to the start of the providers list
func (b *Balancer) ResetProviderList() {
	b.Mu.Lock()
	defer b.Mu.Unlock()

	b.Counter = 0
}
