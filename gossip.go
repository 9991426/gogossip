package main

import (
	"fmt"
	"math/rand"
	"time"
)

// StartGossip sets up the network and starts the gossip algorithms.
func StartGossip(node_num int, infected_num int, should_push bool, should_pull bool) {
	rand.Seed(time.Now().UnixNano())

	// Create the channels for the nodes to communicate with.
	channels := make([]Bichan, node_num)
	for i := 0; i < node_num; i++ {
		channels[i] = Bichan{
			make(chan bool, 1),
			make(chan int, 1),
		}

		// Close the channels when finished, to ensure a clean exit.
		defer close(channels[i].set)
		defer close(channels[i].req)
	}

	// Set up the network with the specified settings.
	network := &Network{
		should_push:  should_push,
		should_pull:  should_pull,
		num_infected: 0,
		saturated:    false,
		channels:     channels,
	}

	// Add nodes to the network, and start their gossip algorithms.
	for i := 0; i < node_num; i++ {
		network.w.Add(1)

		node := Node{
			node_pos: i,
			infected: false,
			network:  network,
		}

		go node.Gossip()
	}

	// Infect the specified number of nodes, and wait time how long it takes for
	// the entire network to get infected.
	start_time := time.Now()
	for i := 0; i < infected_num && i < node_num; i++ {
		channels[i].set <- true
	}

	network.w.Wait()
	duration := time.Since(start_time)

	fmt.Println("Infecting", node_num, "nodes took", duration)
}
