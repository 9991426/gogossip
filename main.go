package main

import (
  "fmt"
  "math/rand"
  "sync"
)

var infected int = 0

func gossip_node(node_pos int, nodes [100]chan int, start chan struct{}, m *sync.Mutex) {
  <-start // wait for all nodes to initialize
  fmt.Println("Initializing node", node_pos)
  status := 0
  rand_pos := rand.Intn(10)

  for {
    x := <- nodes[node_pos]  
    if x == 1 {
      nodes[rand_pos] <- 1
    }

    if status == 0 && x == 1 {
      status = 1
      m.Lock()
      infected += 1
      fmt.Println("Infected Nodes:", infected)
      m.Unlock()
    }

    if infected == 0 {
      return
    }
  }
}

func main() {
  var m sync.Mutex

  start := make(chan struct{})

  var nodes [100]chan int
  for i := 0; i < 100; i++ {
    nodes[i] = make(chan int)
  }

  for i := 0; i < 100; i++ {
    go gossip_node(i, nodes, start, &m)
  }

  close(start)
}