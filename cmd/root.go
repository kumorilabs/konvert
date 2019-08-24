package cmd

import (
	"fmt"
	"os"

	"github.com/ryane/konvert/pkg/konverter"
	"github.com/ryane/konvert/pkg/sources"
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

	rootCmd.Flags().StringVarP(&root.chart, "chart", "c", "", "Helm chart to convert")
	rootCmd.Flags().StringVarP(&root.version, "chart-version", "v", "", "Helm chart version")
	rootCmd.Flags().StringVarP(&root.out, "out", "o", "", "Output directory")

	rootCmd.SetVersionTemplate(`{{.Version}}`)

	return rootCmd
}

func (r *root) run() error {
	if r.chart == "" {
		return fmt.Errorf("chart is required")
	}

	if r.out == "" {
		r.out = "./konvert"
	}

	hs := sources.NewHelmSource(r.chart, r.version)
	converter := konverter.New(hs, r.out)
	return converter.Run()
}

// Execute runs the root command
func Execute(args []string) {
	if err := newRootCommand(args).Execute(); err != nil {
		log.Error(err)
		os.Exit(2)
	}
}
