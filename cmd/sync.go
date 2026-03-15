package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ngavilan-dogfy/woffuk-cli/internal/config"
	gh "github.com/ngavilan-dogfy/woffuk-cli/internal/github"
)

var syncCmd = &cobra.Command{
	Use:   "sync",
	Short: "Re-sync GitHub secrets and workflows from local config",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		password, err := config.GetPassword(cfg.WoffuEmail)
		if err != nil {
			return fmt.Errorf("cannot get password: %w", err)
		}

		fmt.Printf("Syncing secrets to %s...\n", cfg.GithubFork)
		if err := gh.SyncSecrets(cfg, password); err != nil {
			return fmt.Errorf("sync secrets: %w", err)
		}
		fmt.Println("Secrets synced!")

		fmt.Println("Pushing updated workflows...")
		if err := gh.SyncWorkflows(cfg); err != nil {
			return fmt.Errorf("sync workflows: %w", err)
		}
		fmt.Println("Workflows synced!")

		return nil
	},
}
