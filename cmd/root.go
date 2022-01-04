package cmd

import (
	"fmt"
	"os"

	"github.com/kumorilabs/konvert/internal/konvert"
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
	filepath string
}

func newRootCommand(args []string) *cobra.Command {
	root := &root{}
	rootCmd := &cobra.Command{
		Use:   "konvert",
		Short: "konvert generates kustomize bases or kubernetes manifests",
		Long:  `konvert can convert helm charts to kustomize bases or plain kubernetes manifests`,
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
	rootCmd.AddCommand(newFnCommand(args))

	rootCmd.Flags().StringVarP(&root.filepath, "file", "f", "konvert.yaml", "the path to the konvert configuration.")

	return rootCmd
}

func (r *root) run() error {
	return konvert.Konvert(r.filepath)
}

// Execute runs the root command
func Execute(args []string) {
	if err := newRootCommand(args).Execute(); err != nil {
		log.Error(err)
		os.Exit(2)
	}
}
