package cmd

import (
	"github.com/kumorilabs/konvert/internal/functions"
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
)

func newFnCommand() *cobra.Command {
	kp := functions.KonvertProcessor{}
	cmd := command.Build(&kp, command.StandaloneEnabled, false)
	cmd.Use = "fn"
	// TODO: usage
	cmd.Short = "konvert kpt function"
	cmd.Long = "konvert kpt function"
	return cmd
}
