package main

import (
	"math/rand"
	"time"
)

type Node struct {
	node_pos int
	infected bool
	network  *Network
}

// done calls done on the network's waitgroup.
func (n *Node) done() {
	n.network.w.Done()
}

// infect sets the current node to infected, and tells the network to increment
// the number of infected nodes.
func (n *Node) infect() {
	n.infected = true

	n.network.increment_infected()
}

// infect_other attempts to infect other_node. Returns whether the push was
// successful. The push could fail if other_node's set buffer is full.
func (n *Node) infect_other(other_node int) bool {
	dbgPrint(n.node_pos, "->", other_node)
	select {
	case n.network.channels[other_node].set <- n.infected:
		return true
	default:
		return false
	}
}

// request_other attempts to request an infection status from other_node.
// Returns whether the request was successful.
func (n *Node) request_other(other_node int) bool {
	dbgPrint(n.node_pos, "<-", other_node)
	select {
	case n.network.channels[other_node].req <- n.node_pos:
		return true
	default:
		return false
	}
}

// query_set repeatedly reads from the set channel, and infects the current node
// if needed.
func (n *Node) query_set() {
	for {
		infected, ok := <-n.network.channels[n.node_pos].set 

		if !ok {
			break
		}

		dbgPrint(n.node_pos, "<S", infected)
		if !n.infected && infected {
			n.infect()
		}
	}
}

// query_req repeatedly reads from the req channel, and responds with an
// infection set if the current node is infected.
func (n *Node) query_req() {
	for {
		requestor, ok := <-n.network.channels[n.node_pos].req

		if !ok {
			break
		}

		dbgPrint(n.node_pos, "<R", requestor)
		if n.infected {
			n.infect_other(requestor)
		}
	}
}

// Gossip runs the gossip algorithm on the given node with the specified options
// in the network.
func (n *Node) Gossip() {
	defer n.done()
	go n.query_set()
	go n.query_req()

	node_num := len(n.network.channels)

	for {
		if n.network.should_push {
			// If the push is enabled and the node is infected, try infecting a random
			// node.
			if n.infected {
				for {
					rand_pos := rand.Intn(node_num)

					// Generate a random position that is not the same as the current node's.
					if rand_pos == n.node_pos {
						continue
					}

					n.infect_other(rand_pos)
					break
				}
			}
		}

		if n.network.should_pull {
			if !n.infected {
				for {
					rand_pos := rand.Intn(node_num)

					// Generate a random position that is not the same as the current node's.
					if rand_pos == n.node_pos {
						continue
					}

					n.request_other(rand_pos)
					break
				}
			}
		}

		// Exit if the network is fully infected
		if n.network.saturated {
			break
		}

		time.Sleep(time.Millisecond)
	}
}
