package cmd

import (
	"fmt"
	"os"

	"github.com/ryane/konvert/pkg/konverter"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	// GitCommit tracks the current git commit
	GitCommit string
	// Version tracks the current version
	Version string
)

type root struct {
	chart   string
	version string
	out     string
}

func newRootCommand(args []string) *cobra.Command {
	root := &root{}
	rootCmd := &cobra.Command{
		Use:   "konvert",
		Short: "konvert generates kustomize bases",
		Long:  `konvert can convert helm charts and kubernetes manifests to kustomize bases`,
		Run: func(cmd *cobra.Command, args []string) {
			if err := root.run(); err != nil {
				log.Error(err)
				os.Exit(1)
			}
		},
		Version: func() string {
			return fmt.Sprintf("%s (%s)\n", Version, GitCommit)
		}(),
	}

	rootCmd.SetVersionTemplate(`{{.Version}}`)

	return rootCmd
}

func (r *root) run() error {
	// load config
	config, err := konverter.LoadConfig()
	if err != nil {
		return err
	}
	converter := konverter.New(config)
	return converter.Run()
}

// Execute runs the root command
func Execute(args []string) {
	if err := newRootCommand(args).Execute(); err != nil {
		log.Error(err)
		os.Exit(2)
	}
}
