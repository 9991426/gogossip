package main

import (
	"fmt"

	"github.com/integrii/flaggy"
)

var verbose = false

func dbgPrint(a ...interface{}) (n int, err error) {
	if verbose {
		return fmt.Println(a...)
	}

	return 0, nil
}

func main() {
	node_num := 100
	infected_num := 1

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
	flaggy.Bool(&verbose, "v", "verbose", "Print additional transmission information for debugging.")

	flaggy.AttachSubcommand(push_alg, 1)
	flaggy.AttachSubcommand(pull_alg, 1)
	flaggy.AttachSubcommand(pushpull_alg, 1)

	flaggy.DefaultParser.DisableShowVersionWithVersion()
	err := flaggy.DefaultParser.Parse()
	if err != nil {
		flaggy.ShowHelpAndExit(fmt.Sprintf("%v", err))
	}

	should_push := push_alg.Used || pushpull_alg.Used
	should_pull := pull_alg.Used || pushpull_alg.Used

	if !should_push && !should_pull {
		flaggy.ShowHelpAndExit("")
	}

	StartGossip(node_num, infected_num, should_push, should_pull)
}
