package cmd

import (
	"fmt"
	"os"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/ngavilan-dogfy/woffuk-cli/internal/config"
	"github.com/ngavilan-dogfy/woffuk-cli/internal/tui"
	"github.com/ngavilan-dogfy/woffuk-cli/internal/woffu"
)

var rootCmd = &cobra.Command{
	Use:     "woffuk",
	Short:   "Woffu time tracking CLI",
	Version: Version,
	Long: `A CLI tool for Woffu time tracking. Fully scriptable and pipe-friendly.

All commands auto-detect TTY:
  • Terminal  → colored, human-friendly output
  • Piped     → machine-readable TSV
  • --json    → structured JSON

Querying:
  woffuk status              Today's signing status
  woffuk events              Available vacations, hours, etc.
  woffuk schedule            View auto-sign schedule

Actions:
  woffuk sign                Clock in/out right now
  woffuk auto                Check auto-signing status
  woffuk auto on/off         Toggle auto-signing

Configuration:
  woffuk setup               Full setup wizard
  woffuk config              View all settings
  woffuk config edit         Change any individual setting
  woffuk schedule edit       Edit schedule and push to GitHub
  woffuk sync                Re-sync secrets + workflows

Other:
  woffuk update              Update to latest version
  woffuk --version           Show version`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, password, err := loadConfigOrSetup()
		if err != nil {
			return err
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
	rootCmd.AddCommand(configCmd)
	rootCmd.AddCommand(autoCmd)
	rootCmd.AddCommand(updateCmd)
}

// loadConfigOrSetup loads config + password, or guides user to setup.
func loadConfigOrSetup() (*config.Config, string, error) {
	hint := lipgloss.NewStyle().Foreground(lipgloss.Color("245"))

	cfg, err := config.Load()
	if err != nil {
		fmt.Println()
		fmt.Printf("  %s No config found. Run %s to get started.\n\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("!"),
			lipgloss.NewStyle().Bold(true).Render("woffuk setup"))
		fmt.Println(hint.Render("  This is a one-time setup that configures your Woffu credentials,"))
		fmt.Println(hint.Render("  GPS coordinates, and GitHub Actions for auto-signing."))
		fmt.Println()
		return nil, "", fmt.Errorf("run 'woffuk setup' first")
	}

	if cfg.WoffuEmail == "" || cfg.WoffuCompanyURL == "" {
		fmt.Println()
		fmt.Printf("  %s Config is incomplete. Run %s to reconfigure.\n\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("!"),
			lipgloss.NewStyle().Bold(true).Render("woffuk setup"))
		return nil, "", fmt.Errorf("incomplete config — run 'woffuk setup'")
	}

	password, err := config.GetPassword(cfg.WoffuEmail)
	if err != nil {
		fmt.Println()
		fmt.Printf("  %s Password not found in keychain for %s.\n",
			lipgloss.NewStyle().Foreground(lipgloss.Color("214")).Render("!"),
			cfg.WoffuEmail)
		fmt.Printf("  Run %s to reconfigure.\n\n",
			lipgloss.NewStyle().Bold(true).Render("woffuk setup"))
		return nil, "", fmt.Errorf("password not in keychain — run 'woffuk setup'")
	}

	return cfg, password, nil
}
