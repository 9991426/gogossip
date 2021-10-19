package main

import (
	"math/rand"
	"sync"
	"time"
)

// StartGossip sets up the network and starts the gossip algorithms.
func StartGossip(node_num int, infected_num int, should_push bool, should_pull bool, async bool, leader bool) (time.Duration, float64) {

	rand.Seed(time.Now().UnixNano())

	// Create the channels for the nodes to communicate with.
	channels := make([]Bichan, node_num)
	for i := 0; i < node_num; i++ {
		channels[i] = Bichan{
			make(chan bool, 1000),
			make(chan int, 1),
		}

		// Close the channels when finished, to ensure a clean exit.
		defer close(channels[i].set)
		defer close(channels[i].req)
	}

	// Set up the network with the specified settings.
	network := &Network{
		has_leader:   leader,
		async:        !leader && async,
		should_push:  should_push,
		should_pull:  should_pull,
		num_infected: infected_num,
		saturated:    infected_num >= node_num,
		channels:     channels,
	}

	if leader {
		network.w_phase = &sync.WaitGroup{}
	} else {
		network.w_phase = &BulkWaitGroup{}
	}

	network.nodes = make([]Node, node_num)

	// Add nodes to the network, and start their gossip algorithms.
	for i := 0; i < node_num; i++ {
		network.nodes[i] = Node{
			node_pos:   i,
			infected:   i < infected_num,
			stop_phase: make(chan struct{}),
			network:    network,
		}
	}

	// Time how long it takes for the entire network to get infected.
	start_time := time.Now()
	network.Gossip()
	duration := time.Since(start_time)

	total_rounds := 0
	for _, node := range network.nodes {
		total_rounds += node.num_rounds
	}
	avg_rounds := float64(total_rounds) / float64(node_num)

	return duration, avg_rounds
}
