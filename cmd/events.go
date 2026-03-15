package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ngavilan-dogfy/woffuk-cli/internal/config"
	"github.com/ngavilan-dogfy/woffuk-cli/internal/woffu"
)

var eventsCmd = &cobra.Command{
	Use:   "events",
	Short: "Show available events (vacations, hours, etc)",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		password, err := config.GetPassword(cfg.WoffuEmail)
		if err != nil {
			return fmt.Errorf("cannot get password: %w", err)
		}

		client := woffu.NewWoffuClient(cfg.WoffuURL)
		companyClient := woffu.NewCompanyClient(cfg.WoffuCompanyURL)

		token, err := woffu.Authenticate(client, companyClient, cfg.WoffuEmail, password)
		if err != nil {
			return fmt.Errorf("auth failed: %w", err)
		}

		events, err := woffu.GetAvailableEvents(companyClient, token)
		if err != nil {
			return fmt.Errorf("get events: %w", err)
		}

		fmt.Println("Available events:")
		fmt.Println()
		for _, e := range events {
			fmt.Printf("  %-45s %6.0f %s\n", e.Name, e.Available, e.Unit)
		}

		return nil
	},
}
