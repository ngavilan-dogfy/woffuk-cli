package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/gavilanbe/woffuk-cli/internal/config"
	gh "github.com/gavilanbe/woffuk-cli/internal/github"
)

var scheduleCmd = &cobra.Command{
	Use:   "schedule",
	Short: "View or edit auto-sign schedule",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, err := config.Load()
		if err != nil {
			return err
		}

		printSchedule(cfg)
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

		reader := bufio.NewReader(os.Stdin)

		fmt.Println("Edit schedule — enter times as HH:MM separated by commas, or 'off' to disable.")
		fmt.Println("Press Enter to keep current value.")
		fmt.Println()

		cfg.Schedule.Monday = editDay(reader, "Monday", cfg.Schedule.Monday)
		cfg.Schedule.Tuesday = editDay(reader, "Tuesday", cfg.Schedule.Tuesday)
		cfg.Schedule.Wednesday = editDay(reader, "Wednesday", cfg.Schedule.Wednesday)
		cfg.Schedule.Thursday = editDay(reader, "Thursday", cfg.Schedule.Thursday)
		cfg.Schedule.Friday = editDay(reader, "Friday", cfg.Schedule.Friday)

		fmt.Println()
		fmt.Printf("Timezone [%s]: ", cfg.Timezone)
		tz, _ := reader.ReadString('\n')
		tz = strings.TrimSpace(tz)
		if tz != "" {
			cfg.Timezone = tz
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}

		fmt.Println("\nSchedule saved!")
		printSchedule(cfg)

		// Offer to push updated workflows
		if cfg.GithubFork != "" {
			fmt.Printf("\nPush updated workflows to %s? [Y/n]: ", cfg.GithubFork)
			answer, _ := reader.ReadString('\n')
			answer = strings.TrimSpace(strings.ToLower(answer))
			if answer == "" || answer == "y" {
				fmt.Println("Pushing workflows...")
				if err := gh.SyncWorkflows(cfg); err != nil {
					return fmt.Errorf("push workflows: %w", err)
				}
				fmt.Println("Workflows updated!")
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

		fmt.Printf("Pushing workflows to %s...\n", cfg.GithubFork)
		if err := gh.SyncWorkflows(cfg); err != nil {
			return err
		}

		fmt.Println("Workflows updated!")
		return nil
	},
}

func init() {
	scheduleCmd.AddCommand(scheduleEditCmd)
	scheduleCmd.AddCommand(schedulePushCmd)
}

func printSchedule(cfg *config.Config) {
	fmt.Printf("Timezone: %s\n\n", cfg.Timezone)
	printDay("Monday   ", cfg.Schedule.Monday)
	printDay("Tuesday  ", cfg.Schedule.Tuesday)
	printDay("Wednesday", cfg.Schedule.Wednesday)
	printDay("Thursday ", cfg.Schedule.Thursday)
	printDay("Friday   ", cfg.Schedule.Friday)
}

func printDay(name string, day config.DaySchedule) {
	if !day.Enabled {
		fmt.Printf("  %s  off\n", name)
		return
	}
	var times []string
	for _, t := range day.Times {
		times = append(times, t.Time)
	}
	fmt.Printf("  %s  %s\n", name, strings.Join(times, ", "))
}

func editDay(reader *bufio.Reader, name string, current config.DaySchedule) config.DaySchedule {
	currentStr := "off"
	if current.Enabled {
		var times []string
		for _, t := range current.Times {
			times = append(times, t.Time)
		}
		currentStr = strings.Join(times, ", ")
	}

	fmt.Printf("  %s [%s]: ", name, currentStr)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)

	if input == "" {
		return current
	}

	if strings.ToLower(input) == "off" {
		return config.DaySchedule{Enabled: false}
	}

	parts := strings.Split(input, ",")
	var times []config.ScheduleEntry
	for _, p := range parts {
		t := strings.TrimSpace(p)
		if t != "" {
			times = append(times, config.ScheduleEntry{Time: t})
		}
	}

	return config.DaySchedule{Enabled: true, Times: times}
}
