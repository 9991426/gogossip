package main

import (
	"fmt"
	"math"

	"github.com/integrii/flaggy"
)

var verbosity = 0

func dbgPrint(level int, a ...interface{}) (n int, err error) {
	if verbosity >= level {
		fmt.Println(a...)
	}

	return 0, nil
}

type gossipConfig struct {
	node_num     int
	infected_num int
	should_push  bool
	should_pull  bool
	async        bool
	leader       bool
}

type p struct{ a, b bool }

func runBenchmark() {
	configs := make([]gossipConfig, 0, 3*3*4*3)

	//                                sync lead      async         sync nolead
	for _, async_leader := range []p{{false, true}, {true, false}, {false, false}} {
		//                             push           pull           pushpull
		for _, push_pull := range []p{{true, false}, {false, true}, {true, true}} {
			for i := 0; i <= 4; i++ {
				node_num := 2 * int(math.Pow(8, float64(i)))
				for j := 0; j <= 2; j++ {
					infected_num := 1 + (j * node_num / 3)
					configs = append(configs, gossipConfig{node_num, infected_num, push_pull.a, push_pull.b, async_leader.a, async_leader.b})
				}
			}
		}
	}

	for _, push_pull := range []p{{true, false}, {false, true}, {true, true}} {
		for i := 0; i <= 5000; i += 500 {
			node_num := i
			if i == 0 {
				node_num = 2
			}
			configs = append(configs, gossipConfig{node_num, 1, push_pull.a, push_pull.b, true, false})
		}
	}

	fmt.Println(configs)

	for _, c := range configs {
		for i := 0; i < 3; i++ {
			alg := ""
			var network string

			dur, rounds := StartGossip(c.node_num, c.infected_num, c.should_push, c.should_pull, c.async, c.leader)

			if c.should_push {
				alg = "push"
			}
			if c.should_pull {
				alg += "pull"
			}

			if c.async {
				network = "async"
			} else if !c.leader {
				network = "sync"
			} else {
				network = "leader"
			}

			fmt.Printf("%s\t%s\t%d\t%d\t%f\t%f\n", alg, network, c.node_num, c.infected_num, float64(dur.Microseconds())/1000.0, rounds)
		}
	}
}

func main() {
	node_num := 100
	infected_num := 1
	async := false
	leader := false
	verbose := false
	vverbose := false

	benchmark := flaggy.NewSubcommand("bench")
	benchmark.Description = "Benchmark multiple configurations."

	push_alg := flaggy.NewSubcommand("push")
	push_alg.Description = "In each round, each infected node attempts to infect one random node."

	pull_alg := flaggy.NewSubcommand("pull")
	pull_alg.Description = "In each round, each susceptible node attempts to retrieve infection from one random node."

	pushpull_alg := flaggy.NewSubcommand("pushpull")
	pushpull_alg.Description = "In each round, each infected node attempts to infect one random node. Then, each susceptible node attempts to retrieve infection from one random node."

	flaggy.SetName("gogossip")
	flaggy.SetDescription("Gossip simulator")

	flaggy.Int(&node_num, "n", "nodes", "Sets the number of nodes in the network.")
	flaggy.Int(&infected_num, "i", "infected", "Sets the number of initially infected nodes in the network.")
	flaggy.Bool(&async, "a", "async", "Use an asynchronous network.")
	flaggy.Bool(&leader, "l", "leader", "Use a synchronous network with a leader.")
	flaggy.Bool(&verbose, "v", "verbose", "Print additional transmission information for debugging.")
	flaggy.Bool(&vverbose, "vv", "vverbose", "Print more transmission information for debugging.")

	flaggy.AttachSubcommand(benchmark, 1)
	flaggy.AttachSubcommand(push_alg, 1)
	flaggy.AttachSubcommand(pull_alg, 1)
	flaggy.AttachSubcommand(pushpull_alg, 1)

	flaggy.DefaultParser.DisableShowVersionWithVersion()
	err := flaggy.DefaultParser.Parse()
	if err != nil {
		flaggy.ShowHelpAndExit(fmt.Sprintf("%v", err))
	}

	if verbose {
		verbosity = 1
	}
	if vverbose {
		verbosity = 2
	}

	if benchmark.Used {
		runBenchmark()
		return
	}

	should_push := push_alg.Used || pushpull_alg.Used
	should_pull := pull_alg.Used || pushpull_alg.Used

	if !should_push && !should_pull {
		flaggy.ShowHelpAndExit("")
	}

	if leader {
		async = false
	}

	dur, avg_rounds := StartGossip(node_num, infected_num, should_push, should_pull, async, leader)
	fmt.Println("Infecting", node_num, "nodes took", dur, "and avg", avg_rounds, "rounds")
}
