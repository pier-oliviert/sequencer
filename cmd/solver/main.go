package main

import (
	"os"

	"github.com/cert-manager/cert-manager/pkg/acme/webhook/cmd"
	"github.com/pier-oliviert/sequencer/pkg/solver"
)

var GroupName = os.Getenv("GROUP_NAME")
var SolverName = os.Getenv("SOLVER_NAME")
var Namespace = os.Getenv("SEQUENCER_NAMESPACE")

func main() {

	if GroupName == "" {
		panic("GROUP_NAME must be specified")
	}

	if SolverName == "" {
		panic("SOLVER_NAME must be specified")
	}

	if Namespace == "" {
		panic("SEQUENCER_NAMESPACE must be specified")
	}

	cmd.RunWebhookServer(GroupName, solver.New(SolverName, Namespace))
}
