package cmd

import (
	"bufio"
	"fmt"
	"os"
	"strings"

	"github.com/spf13/cobra"

	"github.com/ngavilan-dogfy/woffuk-cli/internal/config"
	"github.com/ngavilan-dogfy/woffuk-cli/internal/notify"
	"github.com/ngavilan-dogfy/woffuk-cli/internal/woffu"
)

var signForce bool

var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "Clock in/out on Woffu (works locally and in CI)",
	Long: `Clock in/out on Woffu. Checks calendar first and only signs on working days.

Examples:
  woffuk sign                Sign for today
  woffuk sign --force        Sign even if not a working day

Batch (from stdin):
  echo "sign" | woffuk sign

In CI, reads credentials from environment variables.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg, password, err := config.LoadOrEnv()
		if err != nil {
			return err
		}

		client := woffu.NewWoffuClient(cfg.WoffuURL)
		companyClient := woffu.NewCompanyClient(cfg.WoffuCompanyURL)

		if isTTY() {
			fmt.Println("Authenticating...")
		}
		token, err := woffu.Authenticate(client, companyClient, cfg.WoffuEmail, password)
		if err != nil {
			return fmt.Errorf("auth failed: %w", err)
		}

		if isTTY() {
			fmt.Println("Checking calendar...")
		}
		info, err := woffu.GetSignInfo(companyClient, token, cfg.Latitude, cfg.Longitude, cfg.HomeLatitude, cfg.HomeLongitude)
		if err != nil {
			return fmt.Errorf("get sign info: %w", err)
		}

		telegramCfg := notify.TelegramConfig{
			BotToken: cfg.Telegram.BotToken,
			ChatID:   cfg.Telegram.ChatID,
		}

		if !info.IsWorkingDay && !signForce {
			if isTTY() {
				fmt.Println("Not a working day — skipping.")
			}
			_ = notify.SendSkippedNotification(telegramCfg, info.Date, "Not a working day")
			return nil
		}

		if isTTY() {
			fmt.Printf("%s %s — signing with coordinates (%.4f, %.4f)\n",
				info.Mode.Emoji(), info.Mode.Label(), info.Latitude, info.Longitude)
		}

		err = woffu.DoSign(companyClient, token, info.Latitude, info.Longitude)
		if err != nil {
			return fmt.Errorf("sign failed: %w", err)
		}

		if isTTY() {
			fmt.Println("Signed successfully!")
		} else {
			fmt.Printf("OK %s %s %s\n", info.Date, info.Mode, info.Mode.Label())
		}

		if err := notify.SendSignedNotification(telegramCfg, info); err != nil && isTTY() {
			fmt.Printf("Warning: telegram notification failed: %s\n", err)
		}

		return nil
	},
}

func init() {
	signCmd.Flags().BoolVar(&signForce, "force", false, "Sign even if not a working day")
}

// readStdinLines reads non-empty lines from stdin (for batch piping).
func readStdinLines() []string {
	var lines []string
	scanner := bufio.NewScanner(os.Stdin)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		lines = append(lines, line)
	}
	return lines
}
