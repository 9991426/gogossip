package main

import (
	"sync"
)

type Bichan struct {
	set chan bool // Set an infected value.
	req chan int  // Node at position is requesting the infected status.
}

type Network struct {
	should_push bool // Whether nodes are allowed to push infected status.
	should_pull bool // Whether nodes are allowed to pull infected status.

	num_infected int        // Guarded by lock. The number of currently infected nodes.
	saturated    bool       // Guarded by lock. Whether the network is fully infected.
	lock         sync.Mutex // The mutex to guard num_infected and saturated.

	channels []Bichan       // The communication channels between nodes.
	w        sync.WaitGroup // The completion WaitGroup.
}

// increment_infected increments the number of infected node in the network.
func (n *Network) increment_infected() {
	n.lock.Lock()

	n.num_infected += 1

	if n.num_infected == len(n.channels) {
		n.saturated = true
	}

	n.lock.Unlock()
}
