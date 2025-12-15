package main

import (
	"fmt"
	"os"

	"orch.io/internal/adapters"
	"orch.io/internal/orchestration"
	"orch.io/pkg/manifest"
	manifestcore "orch.io/pkg/manifest/core"
	"orch.io/pkg/version"

	"github.com/spf13/cobra"
)

func validateCompoments(m *manifestcore.Manifest) error {
	for _, c := range m.Components {
		adapter, err := adapters.Get(c.Type)
		if err != nil {
			return err
		}
		if err := adapter.ValidateComponent(c); err != nil {
			return fmt.Errorf("component %s validation failed: %w", c.Name, err)
		}
	}
	return nil
}

func main() {
	var manifestPath string

	rootCmd := &cobra.Command{
		Use:           "orch",
		Short:         "Orch — ephemeral sandbox orchestrator",
		SilenceUsage:  true,
		SilenceErrors: true,
	}

	rootCmd.PersistentFlags().StringVarP(&manifestPath, "file", "f", "orch.yaml", "Path to manifest")

	upCmd := &cobra.Command{
		Use:   "up",
		Short: "Provision resources defined in manifest",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := manifest.Load(manifestPath)
			if err != nil {
				return err
			}
			if err := validateCompoments(m); err != nil {
				return err
			}
			return orchestration.RunUp(m)
		},
	}

	downCmd := &cobra.Command{
		Use:   "down",
		Short: "Tear down resources from last run",
		RunE: func(cmd *cobra.Command, args []string) error {
			m, err := manifest.Load(manifestPath)
			if err != nil {
				return err
			}
			if err := validateCompoments(m); err != nil {
				return err
			}
			return orchestration.RunDown(m)
		},
	}

	versionCmd := &cobra.Command{
		Use:   "version",
		Short: "Show version information",
		Run: func(cmd *cobra.Command, args []string) {
			fmt.Println(version.String())
		},
	}

	rootCmd.AddCommand(upCmd, downCmd, versionCmd)
	if err := rootCmd.Execute(); err != nil {
		fmt.Println("Error:", err)
		os.Exit(1)
	}
}
