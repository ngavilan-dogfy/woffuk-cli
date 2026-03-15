package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/ngavilan-dogfy/woffuk-cli/internal/config"
	gh "github.com/ngavilan-dogfy/woffuk-cli/internal/github"
)

var autoCmd = &cobra.Command{
	Use:   "auto",
	Short: "View or toggle auto-signing",
	Long:  "Check if GitHub Actions auto-signing is enabled, or turn it on/off.",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		if cfg.GithubFork == "" {
			fmt.Println()
			fmt.Printf("  %s Auto-signing is not set up.\n", sWarn)
			fmt.Printf("  Run %s to configure GitHub Actions.\n\n", sBold.Render("woffuk setup"))
			return nil
		}

		return showAutoStatus(cfg.GithubFork)
	},
}

var autoOnCmd = &cobra.Command{
	Use:   "on",
	Short: "Enable auto-signing",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if cfg.GithubFork == "" {
			fmt.Printf("\n  %s Run %s first.\n\n", sWarn, sBold.Render("woffuk setup"))
			return nil
		}

		var enableErr error
		spinner.New().
			Title("Enabling auto-sign...").
			Action(func() { enableErr = gh.EnableAutoSign(cfg.GithubFork) }).
			Run()

		if enableErr != nil {
			fmt.Printf("\n  %s Could not enable: %s\n\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗"), enableErr)
			return nil
		}

		fmt.Printf("\n  %s Auto-signing enabled\n\n", sOk)
		return showAutoStatus(cfg.GithubFork)
	},
}

var autoOffCmd = &cobra.Command{
	Use:   "off",
	Short: "Disable auto-signing",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}
		if cfg.GithubFork == "" {
			fmt.Printf("\n  %s Run %s first.\n\n", sWarn, sBold.Render("woffuk setup"))
			return nil
		}

		var confirm bool
		huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title("Disable auto-signing?").
					Description("Woffu will no longer be clocked automatically").
					Affirmative("Disable").
					Negative("Cancel").
					Value(&confirm),
			),
		).Run()

		if !confirm {
			return nil
		}

		var disableErr error
		spinner.New().
			Title("Disabling auto-sign...").
			Action(func() { disableErr = gh.DisableAutoSign(cfg.GithubFork) }).
			Run()

		if disableErr != nil {
			fmt.Printf("\n  %s Could not disable: %s\n\n",
				lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Render("✗"), disableErr)
			return nil
		}

		fmt.Printf("\n  %s Auto-signing disabled\n\n", sOk)
		return nil
	},
}

func init() {
	autoCmd.AddCommand(autoOnCmd)
	autoCmd.AddCommand(autoOffCmd)
}

func showAutoStatus(repo string) error {
	var workflows []gh.WorkflowStatus
	var statusErr error

	spinner.New().
		Title("Checking workflows...").
		Action(func() { workflows, statusErr = gh.GetAutoSignStatus(repo) }).
		Run()

	if statusErr != nil {
		fmt.Printf("\n  %s Could not check status: %s\n\n", sWarn, statusErr)
		return nil
	}

	sActive := lipgloss.NewStyle().Foreground(lipgloss.Color("82")).Bold(true)
	sDisabled := lipgloss.NewStyle().Foreground(lipgloss.Color("196")).Bold(true)
	sName := lipgloss.NewStyle().Width(20)

	fmt.Printf("\n  Repo: %s\n\n", sBold.Render(repo))

	for _, w := range workflows {
		status := sActive.Render("active")
		if w.State != "active" {
			status = sDisabled.Render("disabled")
		}
		fmt.Printf("  %s %s\n", sName.Render(w.Name), status)
	}

	fmt.Println()
	fmt.Printf("  Toggle: %s / %s\n\n",
		sBold.Render("woffuk auto on"),
		sBold.Render("woffuk auto off"))

	return nil
}
