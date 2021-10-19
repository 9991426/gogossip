package main

import (
	"sync"
	"time"
)

type Bichan struct {
	set chan bool // Set an infected value.
	req chan int  // Node at position is requesting the infected status.
}

type WaitGroupLike interface {
	Add(delta int)
	Done()
	Wait()
}

// Type guards
var _ WaitGroupLike = &BulkWaitGroup{}
var _ WaitGroupLike = &sync.WaitGroup{}

type Network struct {
	has_leader bool // Whether this network has a leader.
	async      bool // Whether the network is asynchronous.

	should_push bool // Whether nodes are allowed to push infected status.
	should_pull bool // Whether nodes are allowed to pull infected status.

	num_infected int          // Guarded by lock. The number of currently infected nodes.
	saturated    bool         // Guarded by lock. Whether the network is fully infected.
	lock         sync.RWMutex // The mutex to guard num_infected and saturated.

	nodes    []Node         // The nodes in the network.
	channels []Bichan       // The communication channels between nodes.
	w        sync.WaitGroup // The completion WaitGroup.
	w_phase  WaitGroupLike  // The phase synchronizer.
}

func (n *Network) Gossip() {
	node_num := len(n.nodes)

	if n.has_leader {
		num_rounds := 0
		var starttime time.Time
		var pushdur time.Duration = 0
		var pushcdur time.Duration = 0
		var pulldur time.Duration = 0
		var pullcdur time.Duration = 0

		for {
			num_rounds += 1

			if n.should_push {
				dbgPrint(2, "start push")

				// Save the infected value to the current phase infected value.
				for i := range n.nodes {
					node := &n.nodes[i]
					node.phase_infected = node.infected
					go node.query_set()
				}

				// Push infection to other nodes.
				starttime = time.Now()

				n.w_phase.Add(node_num)
				for i := range n.nodes {
					node := &n.nodes[i]
					go node.infect_rand(node_num)
				}
				dbgPrint(2, "wait a")
				n.w_phase.Wait()

				pushdur += time.Since(starttime)
				starttime = time.Now()

				// Clean up the channels.
				n.w_phase.Add(node_num)
				for i := range n.nodes {
					node := &n.nodes[i]
					go node.cleanup(node_num, true)
				}
				dbgPrint(2, "wait cleanup")
				n.w_phase.Wait()

				pushcdur += time.Since(starttime)
			}

			if n.should_pull {
				// Push the current infected value onto the set channel. This will be
				// replaced each time it is read.
				dbgPrint(2, n.nodes)
				for i := range n.nodes {
					node := &n.nodes[i]
					n.channels[node.node_pos].set <- node.infected
					dbgPrint(2, node.node_pos, len(n.channels[node.node_pos].set))
				}

				// Pull infection from other nodes.
				starttime = time.Now()

				n.w_phase.Add(node_num)
				for i := range n.nodes {
					node := &n.nodes[i]
					go node.request_rand(node_num)
				}
				n.w_phase.Wait()

				pulldur += time.Since(starttime)
				starttime = time.Now()

				// Clean up the channels.
				n.w_phase.Add(node_num)
				for i := range n.nodes {
					node := &n.nodes[i]
					go node.cleanup(node_num, false)
				}
				n.w_phase.Wait()

				pullcdur += time.Since(starttime)
			}

			// Exit if the network is fully infected
			n.lock.RLock()
			if n.saturated {
				n.lock.RUnlock()
				break
			}
			n.lock.RUnlock()
		}

		dbgPrint(1, "push", pushdur)
		dbgPrint(1, "push clean", pushcdur)
		dbgPrint(1, "pull", pulldur)
		dbgPrint(1, "pull clean", pullcdur)
		n.nodes[0].num_rounds = num_rounds * node_num
	} else {
		node_num := len(n.nodes)
		n.w.Add(node_num)

		for i := 0; i < node_num; i++ {
			go n.nodes[i].Gossip()
		}

		n.w.Wait()
	}
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
