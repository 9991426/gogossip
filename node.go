package main

import (
	"math/rand"
	"time"
)

type Node struct {
	node_pos       int
	infected       bool
	phase_infected bool
	stop_phase     chan struct{}
	num_rounds     int
	network        *Network
}

// done calls done on the network's waitgroup.
func (n *Node) done() {
	n.network.w.Done()
}

// infect sets the current node to infected, and tells the network to increment
// the number of infected nodes.
func (n *Node) infect(infected bool) {
	if n.infected || !infected {
		return
	}

	dbgPrint(1, n.node_pos, "I")
	n.infected = true

	n.network.increment_infected()
}

func (n *Node) infect_rand(node_num int) {
	if !n.network.async {
		defer n.network.w_phase.Done()
	}

	if n.infected {
		rand_pos := n.node_pos

		// Generate a random position that is not the same as the current node's.
		for rand_pos == n.node_pos {
			rand_pos = rand.Intn(node_num)
		}

		if n.network.async {
			n.infect_other_async(rand_pos)
		} else {
			n.infect_other_sync(rand_pos)
		}
	}
}

func (n *Node) request_rand(node_num int) {
	if !n.network.async {
		defer n.network.w_phase.Done()
	}

	if !n.infected {
		rand_pos := n.node_pos

		// Generate a random position that is not the same as the current node's.
		for rand_pos == n.node_pos {
			rand_pos = rand.Intn(node_num)
		}

		if n.network.async {
			n.request_other_async(rand_pos)
		} else {
			n.request_other_sync(rand_pos)
		}
	}
}

// infect_other_sync pushes its phase infection status to other_node.
func (n *Node) infect_other_sync(other_node int) bool {
	dbgPrint(1, n.node_pos, "->", other_node)
	n.network.channels[other_node].set <- n.phase_infected
	return true
}

// request_other_sync pulls other_node's infection status and immediately
// replaces it for other readers.
func (n *Node) request_other_sync(other_node int) bool {
	dbgPrint(1, n.node_pos, "<-", other_node)
	dbgPrint(2, other_node, len(n.network.channels[other_node].set))
	infected := <-n.network.channels[other_node].set
	dbgPrint(2, other_node, len(n.network.channels[other_node].set))
	n.network.channels[other_node].set <- infected
	dbgPrint(2, other_node, len(n.network.channels[other_node].set))

	n.infect(infected)

	return true
}

// infect_other_async attempts to infect other_node. Returns whether the push was
// successful. The push could fail if other_node's set buffer is full.
func (n *Node) infect_other_async(other_node int) bool {
	dbgPrint(1, n.node_pos, "->", other_node)
	select {
	case n.network.channels[other_node].set <- n.infected:
		return true
	default:
		return false
	}
}

// request_other_sync attempts to request an infection status from other_node.
// Returns whether the request was successful.
func (n *Node) request_other_async(other_node int) bool {
	dbgPrint(1, n.node_pos, "<-", other_node)
	select {
	case n.network.channels[other_node].req <- n.node_pos:
		return true
	default:
		return false
	}
}

// query_set repeatedly reads from the set channel, and infects the current node
// if needed. Used in either sync or async.
func (n *Node) query_set() {
	dbgPrint(2, n.node_pos, "start query")
	for {
		select {
		case <-n.stop_phase:
			dbgPrint(2, n.node_pos, "stop query")
			return
		case infected, ok := <-n.network.channels[n.node_pos].set:
			if !ok {
				return
			}

			dbgPrint(1, n.node_pos, "<S", infected)
			n.infect(infected)
		}
	}
}

// query_req repeatedly reads from the req channel, and responds with an
// infection set if the current node is infected. Used only in async.
func (n *Node) query_req() {
	for {
		requestor, ok := <-n.network.channels[n.node_pos].req

		if !ok {
			break
		}

		dbgPrint(1, n.node_pos, "<R", requestor)
		if n.infected {
			n.infect_other_async(requestor)
		}
	}
}

// cleanup clears stops running phase handlers and clears the buffer of the set
// channel. Used only in sync.
func (n *Node) cleanup(node_num int, stop bool) {
	defer n.network.w_phase.Done()
	dbgPrint(2, n.node_pos, "stop phase")

	if stop {
		n.stop_phase <- struct{}{}
	}

	for {
		select {
		case <-n.network.channels[n.node_pos].set:
		default:
			return
		}
	}
}

// Gossip runs the gossip algorithm on the given node with the specified options
// in the network.
func (n *Node) Gossip() {
	defer n.done()

	node_num := len(n.network.channels)
	async := n.network.async
	var starttime time.Time
	var pushdur time.Duration = 0
	var pushcdur time.Duration = 0
	var pulldur time.Duration = 0
	var pullcdur time.Duration = 0

	if async {
		go n.query_set()
		go n.query_req()
	}

	for {
		n.num_rounds += 1

		if n.network.should_push {
			// If the push is enabled and the node is infected, try infecting a random
			// node.
			if async {
				n.infect_rand(node_num)
			} else {
				dbgPrint(2, n.node_pos, "start push")

				// Save the infected value to the current phase infected value.
				n.phase_infected = n.infected

				// Add the number of nodes to the waitgroup.
				n.network.w_phase.Add(node_num)

				// Push infection to other nodes.
				if n.node_pos == 0 {
					starttime = time.Now()
				}
				go n.query_set()
				go n.infect_rand(node_num)

				dbgPrint(2, n.node_pos, "wait a")
				n.network.w_phase.Wait()
				if n.node_pos == 0 {
					pushdur += time.Since(starttime)
					starttime = time.Now()
				}

				// Clean up the channels.
				n.network.w_phase.Add(node_num)
				go n.cleanup(node_num, true)
				dbgPrint(2, n.node_pos, "wait cleanup")
				n.network.w_phase.Wait()
				if n.node_pos == 0 {
					pushcdur += time.Since(starttime)
					starttime = time.Now()
				}
			}
		}

		if n.network.should_pull {
			if async {
				n.request_rand(node_num)
			} else {
				dbgPrint(2, n.node_pos, "start pull")

				// Push the current infected value onto the set channel. This will be
				// replaced each time it is read.
				n.network.channels[n.node_pos].set <- n.infected

				// Add the number of nodes to the waitgroup.
				n.network.w_phase.Add(node_num)

				// Pull infection from other nodes.
				if n.node_pos == 0 {
					starttime = time.Now()
				}
				go n.request_rand(node_num)

				dbgPrint(2, n.node_pos, "wait d")
				n.network.w_phase.Wait()
				if n.node_pos == 0 {
					pulldur += time.Since(starttime)
					starttime = time.Now()
				}

				// Clean up the channels.
				n.network.w_phase.Add(node_num)
				go n.cleanup(node_num, false)
				dbgPrint(2, n.node_pos, "wait cleanup")
				n.network.w_phase.Wait()
				if n.node_pos == 0 {
					pullcdur += time.Since(starttime)
					starttime = time.Now()
				}
			}
		}

		if async {
			time.Sleep(time.Millisecond)
		}

		// Exit if the network is fully infected
		n.network.lock.RLock()
		if n.network.saturated {
			dbgPrint(2, n.node_pos, "FULLY SATURATED")
			n.network.lock.RUnlock()
			break
		}
		n.network.lock.RUnlock()
	}

	if n.node_pos == 0 {
		dbgPrint(1, "push", pushdur)
		dbgPrint(1, "push clean", pushcdur)
		dbgPrint(1, "pull", pulldur)
		dbgPrint(1, "pull clean", pullcdur)
	}
}
