package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/spf13/cobra"
	"golang.org/x/term"

	"github.com/ngavilan-dogfy/woffuk-cli/internal/config"
	"github.com/ngavilan-dogfy/woffuk-cli/internal/geocode"
	gh "github.com/ngavilan-dogfy/woffuk-cli/internal/github"
)

var setupCmd = &cobra.Command{
	Use:   "setup",
	Short: "Interactive setup wizard",
	RunE: func(cmd *cobra.Command, args []string) error {
		reader := bufio.NewReader(os.Stdin)

		fmt.Println("=== woffuk setup ===")
		fmt.Println()

		// --- Woffu credentials ---

		email := prompt(reader, "Woffu email")

		fmt.Print("Woffu password: ")
		passBytes, err := term.ReadPassword(int(os.Stdin.Fd()))
		if err != nil {
			return fmt.Errorf("read password: %w", err)
		}
		password := string(passBytes)
		fmt.Println()

		company := prompt(reader, "Company name (e.g. dogfydiet)")
		companyURL := "https://" + company + ".woffu.com"
		fmt.Printf("  -> %s\n", companyURL)

		// --- Locations ---

		fmt.Println()
		fmt.Println("=== Office location ===")
		fmt.Println()
		officeLat, officeLon, err := promptLocation(reader, "Where is your office?")
		if err != nil {
			return err
		}

		// Nominatim rate limit
		time.Sleep(time.Second)

		fmt.Println()
		fmt.Println("=== Home location ===")
		fmt.Println()
		homeLat, homeLon, err := promptLocation(reader, "Where is your home?")
		if err != nil {
			return err
		}

		// --- Schedule ---

		fmt.Println()
		fmt.Println("=== Auto-sign schedule ===")
		fmt.Println()

		zone, _ := time.Now().Zone()
		tz := prompt(reader, fmt.Sprintf("Timezone [%s]", zone))
		if tz == "" {
			tz = zone
		}

		fmt.Println()
		fmt.Println("Default schedule:")
		fmt.Println("  Mon-Thu: 08:30, 13:30, 14:15, 17:30")
		fmt.Println("  Fri:     08:00, 15:00")
		fmt.Println()

		useDefault := prompt(reader, "Use default schedule? [Y/n]")
		schedule := config.DefaultSchedule()

		if strings.ToLower(useDefault) == "n" {
			fmt.Println()
			fmt.Println("Enter times as HH:MM separated by commas, or 'off' to disable a day.")
			fmt.Println()
			schedule.Monday = promptDay(reader, "Monday", schedule.Monday)
			schedule.Tuesday = promptDay(reader, "Tuesday", schedule.Tuesday)
			schedule.Wednesday = promptDay(reader, "Wednesday", schedule.Wednesday)
			schedule.Thursday = promptDay(reader, "Thursday", schedule.Thursday)
			schedule.Friday = promptDay(reader, "Friday", schedule.Friday)
		}

		// --- Telegram (optional) ---

		fmt.Println()
		fmt.Println("=== Telegram notifications (optional) ===")
		fmt.Println()
		telegramToken := prompt(reader, "Telegram Bot Token (Enter to skip)")
		var telegramCfg config.TelegramConfig
		if telegramToken != "" {
			telegramChatID := prompt(reader, "Telegram Chat ID")
			telegramCfg = config.TelegramConfig{
				BotToken: telegramToken,
				ChatID:   telegramChatID,
			}
			fmt.Println("  Telegram notifications enabled")
		} else {
			fmt.Println("  Skipped")
		}

		// --- Save ---

		cfg := &config.Config{
			WoffuURL:        "https://app.woffu.com/api",
			WoffuCompanyURL: companyURL,
			WoffuEmail:      email,
			Latitude:        officeLat,
			Longitude:       officeLon,
			HomeLatitude:    homeLat,
			HomeLongitude:   homeLon,
			Timezone:        tz,
			Schedule:        schedule,
			Telegram:        telegramCfg,
		}

		if err := config.Save(cfg); err != nil {
			return fmt.Errorf("save config: %w", err)
		}
		fmt.Println("\nConfig saved to ~/.woffuk.yaml")

		if err := config.SetPassword(email, password); err != nil {
			return fmt.Errorf("save password to keychain: %w", err)
		}
		fmt.Println("Password saved to OS keychain")

		// --- GitHub ---

		fmt.Println()
		forkAnswer := prompt(reader, "Fork repo and configure GitHub Actions? [Y/n]")
		if forkAnswer == "" || strings.ToLower(forkAnswer) == "y" {
			fmt.Println("  Forking repo...")
			forkName, err := gh.ForkAndSetup(cfg, password)
			if err != nil {
				return fmt.Errorf("github setup: %w", err)
			}
			cfg.GithubFork = forkName
			if err := config.Save(cfg); err != nil {
				return fmt.Errorf("save config with fork: %w", err)
			}
			fmt.Printf("  Fork: %s\n", forkName)
			fmt.Println("  Secrets configured")
			fmt.Println("  Workflows generated and pushed")
			fmt.Println("  GitHub Actions enabled")
		}

		fmt.Println()
		fmt.Println("Setup complete! Auto-signing is active.")
		printSetupSchedule(schedule)
		fmt.Println()
		fmt.Println("Run 'woffuk' to open the dashboard.")

		return nil
	},
}

// promptLocation handles the interactive location search with multiple results.
func promptLocation(reader *bufio.Reader, question string) (float64, float64, error) {
	for {
		query := prompt(reader, question)
		if query == "" {
			continue
		}

		fmt.Println("  Searching...")
		results, err := geocode.Search(query, 5)
		if err != nil {
			fmt.Printf("  Error: %s\n", err)
			fmt.Println("  Try again with a different query.")
			continue
		}

		if len(results) == 0 {
			fmt.Println("  No results found. Try adding more details (city, country).")
			continue
		}

		if len(results) == 1 {
			r := results[0]
			fmt.Printf("  Found: %s\n", r.DisplayName)
			fmt.Printf("  Coordinates: %.6f, %.6f\n", r.Lat, r.Lon)
			confirm := prompt(reader, "  Is this correct? [Y/n]")
			if confirm == "" || strings.ToLower(confirm) == "y" {
				return r.Lat, r.Lon, nil
			}
			fmt.Println("  Try again with a different query.")
			continue
		}

		// Multiple results — let the user pick
		fmt.Println()
		for i, r := range results {
			fmt.Printf("  %d) %s\n", i+1, r.DisplayName)
		}
		fmt.Println("  0) None of these — search again")
		fmt.Println()

		choice := prompt(reader, "  Pick a number")
		n, err := strconv.Atoi(strings.TrimSpace(choice))
		if err != nil || n < 0 || n > len(results) {
			fmt.Println("  Invalid choice. Try again.")
			continue
		}

		if n == 0 {
			continue
		}

		r := results[n-1]
		fmt.Printf("  Selected: %s\n", r.DisplayName)
		fmt.Printf("  Coordinates: %.6f, %.6f\n", r.Lat, r.Lon)
		return r.Lat, r.Lon, nil
	}
}

func prompt(reader *bufio.Reader, label string) string {
	fmt.Printf("%s: ", label)
	input, _ := reader.ReadString('\n')
	return strings.TrimSpace(input)
}

func promptDay(reader *bufio.Reader, name string, current config.DaySchedule) config.DaySchedule {
	var currentTimes []string
	for _, t := range current.Times {
		currentTimes = append(currentTimes, t.Time)
	}
	defaultStr := strings.Join(currentTimes, ", ")

	fmt.Printf("  %s [%s]: ", name, defaultStr)
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

func printSetupSchedule(s config.Schedule) {
	days := []struct {
		name string
		day  config.DaySchedule
	}{
		{"Mon", s.Monday},
		{"Tue", s.Tuesday},
		{"Wed", s.Wednesday},
		{"Thu", s.Thursday},
		{"Fri", s.Friday},
	}

	for _, d := range days {
		if !d.day.Enabled {
			fmt.Printf("  %s: off\n", d.name)
			continue
		}
		var times []string
		for _, t := range d.day.Times {
			times = append(times, t.Time)
		}
		fmt.Printf("  %s: %s\n", d.name, strings.Join(times, ", "))
	}
}
