package main

import (
	"flag"
	"fmt"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"
)

const DBG = false

func dbgPrint(a ...interface{}) (n int, err error) {
	if (DBG) {
		return fmt.Println(a...)
	}

	return 0, nil
}

type Bichan struct {
	set chan bool // Set an infected value.
	req chan int  // Node at position is requesting the infected status.
}

type Network struct {
	should_push bool // Whether nodes are allowed to push infected status.
	should_pull bool // Whether nodes are allowed to pull infected status.

	num_infected int         // Guarded by lock. The number of currently infected nodes.
	saturated    bool        // Guarded by lock. Whether the network is fully infected.
	lock         sync.Mutex // The mutex to guard num_infected and saturated.

	channels []Bichan        // The communication channels between nodes.
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
		if ok {
			dbgPrint(n.node_pos, "<S", infected)
			if !n.infected && infected {
				n.infect()
			}
		}
	}
}

// query_req repeatedly reads from the req channel, and responds with an
// infection set if the current node is infected.
func (n *Node) query_req() {
	for {
		requestor, ok := <-n.network.channels[n.node_pos].req
		if ok {
			dbgPrint(n.node_pos, "<R", requestor)
			if n.infected {
				n.infect_other(requestor)
			}
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

	// Close all the channels, to tell the query goroutines to stop.
	for _, ch := range channels {
		close(ch.set)
		close(ch.req)
	}

	fmt.Println("Infecting", node_num, "nodes took", duration)
}

func main() {
	node_num := flag.Int("n", 100, "Sets the number of nodes in the network.")
	infected_num := flag.Int("i", 1, "Sets the number of infected nodes in the network.")

	flag.Usage = func() {
		fmt.Println("Usage:")
		fmt.Printf("%s [-n node_num] [-i infected_num] {algorithm}\n", os.Args[0])
		flag.PrintDefaults()
		fmt.Println("  algorithm")
		fmt.Println("    \tMUST BE LAST. The gossip algorithm to run. Can be push, pull, or pushpull.")
		os.Exit(2)
	}

	flag.Parse()

	remaining := flag.Args()

	should_push := false
	should_pull := false

	if flag.NArg() != 1 {
		flag.Usage()
	} else {
		switch strings.ToLower(remaining[0]) {
		case "push":
			should_push = true
		case "pull":
			should_pull = true
		case "pushpull":
			should_push = true
			should_pull = true
		default:
			flag.Usage()
		}
	}

	StartGossip(*node_num, *infected_num, should_push, should_pull)
}
