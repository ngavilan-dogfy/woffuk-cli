package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/spf13/cobra"

	"github.com/gavilanbe/woffuk-cli/internal/config"
	"github.com/gavilanbe/woffuk-cli/internal/tui"
	"github.com/gavilanbe/woffuk-cli/internal/woffu"
)

var rootCmd = &cobra.Command{
	Use:   "woffuk",
	Short: "Woffu time tracking CLI",
	Long:  "CLI tool to automatically clock in/out of Woffu with an interactive TUI dashboard.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		password, err := config.GetPassword(cfg.WoffuEmail)
		if err != nil {
			return fmt.Errorf("cannot get password from keychain: %w\nRun 'woffuk setup' to configure", err)
		}

		client := woffu.NewWoffuClient(cfg.WoffuURL)
		companyClient := woffu.NewCompanyClient(cfg.WoffuCompanyURL)

		model := tui.NewDashboard(client, companyClient, cfg, password)
		p := tea.NewProgram(model, tea.WithAltScreen())
		if _, err := p.Run(); err != nil {
			return err
		}
		return nil
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(signCmd)
	rootCmd.AddCommand(statusCmd)
	rootCmd.AddCommand(eventsCmd)
	rootCmd.AddCommand(setupCmd)
	rootCmd.AddCommand(syncCmd)
	rootCmd.AddCommand(scheduleCmd)
}
