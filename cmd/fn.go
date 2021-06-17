package cmd

import (
	"github.com/spf13/cobra"
	"sigs.k8s.io/kustomize/kyaml/fn/framework"
	"sigs.k8s.io/kustomize/kyaml/fn/framework/command"
	"sigs.k8s.io/kustomize/kyaml/kio/filters"
)

type Konvert struct {
	Spec KonvertSpec `yaml:"spec,omitempty"`
}

type KonvertSpec struct {
	Name string `yaml:"name,omitempty"`
}

func newFnCommand(args []string) *cobra.Command {
	konvert := &Konvert{}
	sp := framework.SimpleProcessor{
		Config: konvert,
		Filter: filters.FormatFilter{
			UseSchema: false,
		},
	}
	cmd := command.Build(&sp, command.StandaloneEnabled, false)
	cmd.Use = "fn"
	cmd.Short = "konvert kpt function"
	cmd.Long = "konvert kpt function"
	return cmd
}
