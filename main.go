package main

import (
  // "fmt"
  "math/rand"
  "sync"
  "time"
)

var num_infected int = 0
const node_num int = 2500

func gossip_push(node_pos int, nodes *[node_num]chan bool, start chan struct{}, m *sync.Mutex, w *sync.WaitGroup) {
  _, _ = <-start // wait for all nodes to initialize
  infected := false
  
  for {
    if infected {
      rand_pos := node_pos
      for rand_pos == node_pos {
        rand_pos = rand.Intn(node_num)
      }

      select {
        case nodes[rand_pos] <- infected:
        default:
      }
    }

    select {
      case new_infected := <- nodes[node_pos]:
        if !infected && new_infected {
          infected = new_infected
          m.Lock()
          num_infected += 1
          m.Unlock()
        }
      default:
    }

    if num_infected == node_num {
      break
    }

    time.Sleep(time.Millisecond)
  }

  w.Done()
}

func main() {
  var m sync.Mutex
  var w sync.WaitGroup

  start := make(chan struct{})

  var nodes [node_num]chan bool
  for i := 0; i < node_num; i++ {
    nodes[i] = make(chan bool, 1)
  }

  for i := 0; i < node_num; i++ {
    w.Add(1)
    go gossip_push(i, &nodes, start, &m, &w)
  }

  close(start)

  nodes[0] <- true

  w.Wait()
}
