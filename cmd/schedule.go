package cmd

import (
	"fmt"

	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/huh/spinner"
	"github.com/charmbracelet/lipgloss"
	"github.com/spf13/cobra"

	"github.com/ngavilan-dogfy/woffux/internal/config"
	gh "github.com/ngavilan-dogfy/woffux/internal/github"
)

var scheduleJSONFlag bool

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "View or edit auto-sign schedule",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		// JSON output
		if scheduleJSONFlag {
			return printJSON(scheduleToJSON(cfg))
		}

		sIn := lipgloss.NewStyle().Foreground(lipgloss.Color("82"))
		sOut := lipgloss.NewStyle().Foreground(lipgloss.Color("196"))

		fmt.Printf("Timezone: %s\n\n", cfg.Timezone)
		fmt.Printf("  %s = clock in    %s = clock out\n\n", sIn.Render("▶ IN"), sOut.Render("■ OUT"))
		printScheduleVisual(cfg.Schedule, sIn, sOut)
		fmt.Println()
		return nil
	},
}

var scheduleEditCmd = &cobra.Command{
	Use:   "edit",
	Short: "Edit auto-sign schedule interactively",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		schedule, tz, err := scheduleWizard()
		if err != nil {
			return err
		}

		cfg.Schedule = schedule
		if tz != "" {
			cfg.Timezone = tz
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Printf("  %s Schedule saved!\n", sOk)

		// Push to GitHub if configured
		if cfg.GithubFork != "" {
			var push bool
			huh.NewForm(
				huh.NewGroup(
					huh.NewConfirm().
						Title(fmt.Sprintf("Push to %s?", cfg.GithubFork)).
						Affirmative("Yes").
						Negative("Skip").
						Value(&push),
				),
			).Run()

			if push {
				var pushErr error
				spinner.New().
					Title("Pushing workflows...").
					Action(func() { pushErr = gh.SyncWorkflows(cfg) }).
					Run()

				if pushErr != nil {
					fmt.Printf("  %s Push failed: %s\n", sWarn, pushErr)
				} else {
					fmt.Printf("  %s Workflows updated!\n", sOk)
				}
			}
		}

		return nil
	},
}

var schedulePushCmd = &cobra.Command{
	Use:   "push",
	Short: "Push current schedule as GitHub Actions workflows",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		var pushErr error
		spinner.New().
			Title(fmt.Sprintf("Pushing to %s...", cfg.GithubFork)).
			Action(func() { pushErr = gh.SyncWorkflows(cfg) }).
			Run()

		if pushErr != nil {
			return pushErr
		}
		fmt.Printf("  %s Workflows updated!\n", sOk)
		return nil
	},
}

func init() {
	scheduleCmd.Flags().BoolVar(&scheduleJSONFlag, "json", false, "Output as JSON")
	scheduleCmd.AddCommand(scheduleEditCmd)
	scheduleCmd.AddCommand(schedulePushCmd)
}

// scheduleToJSON builds a structured map for JSON output.
func scheduleToJSON(cfg *config.Config) map[string]interface{} {
	result := map[string]interface{}{
		"timezone": cfg.Timezone,
		"days":     daySchedulesToJSON(cfg.Schedule),
	}
	if cfg.ActiveSchedule != "" {
		result["active_preset"] = cfg.ActiveSchedule
	}
	return result
}

func daySchedulesToJSON(s config.Schedule) map[string]interface{} {
	return map[string]interface{}{
		"monday":    dayToJSON(s.Monday),
		"tuesday":   dayToJSON(s.Tuesday),
		"wednesday": dayToJSON(s.Wednesday),
		"thursday":  dayToJSON(s.Thursday),
		"friday":    dayToJSON(s.Friday),
	}
}

func dayToJSON(d config.DaySchedule) map[string]interface{} {
	result := map[string]interface{}{
		"enabled": d.Enabled,
	}
	if d.Enabled {
		var times []string
		for _, t := range d.Times {
			times = append(times, t.Time)
		}
		result["times"] = times
	}
	return result
}
