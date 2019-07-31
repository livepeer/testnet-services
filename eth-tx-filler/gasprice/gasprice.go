package gasprice

import (
	"crypto/rand"
	"log"
	"math/big"
	"sync"
	"time"
)

// Randomizer randomly calculates a new gas price within a range at regular intervals
type Randomizer struct {
	randomizeInterval time.Duration
	maxGasPrice       *big.Int
	minGasPrice       *big.Int

	running bool

	mu       sync.RWMutex
	gasPrice *big.Int

	quit chan struct{}
}

// NewRandomizer returns a Randomizer instance
func NewRandomizer(interval time.Duration, max *big.Int, min *big.Int) *Randomizer {
	return &Randomizer{
		randomizeInterval: interval,
		maxGasPrice:       max,
		minGasPrice:       min,
		gasPrice:          min,
		quit:              make(chan struct{}),
	}
}

// Start initiates the gas price randomize loop
func (r *Randomizer) Start() {
	if r.running {
		return
	}

	go r.startRandomizeLoop()

	r.running = true
}

// Stop signals the gas price randomize loop to exit gracefully
func (r *Randomizer) Stop() {
	if !r.running {
		return
	}

	close(r.quit)
}

// GasPrice returns the current gas price
func (r *Randomizer) GasPrice() *big.Int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	return r.gasPrice
}

func (r *Randomizer) randomizeGasPrice() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	diff := new(big.Int).Sub(r.maxGasPrice, r.minGasPrice)

	n, err := rand.Int(rand.Reader, diff)
	if err != nil {
		return err
	}

	r.gasPrice = new(big.Int).Add(n, r.minGasPrice)

	log.Printf("new gas price = %v", r.gasPrice)

	return nil
}

func (r *Randomizer) startRandomizeLoop() {
	ticker := time.NewTicker(r.randomizeInterval)

	for {
		select {
		case <-ticker.C:
			if err := r.randomizeGasPrice(); err != nil {
				log.Printf("error randomizing gas price: %v", err)
			}
		case <-r.quit:
			return
		}
	}
}
