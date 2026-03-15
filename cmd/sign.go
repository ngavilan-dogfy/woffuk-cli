package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/gavilanbe/woffuk-cli/internal/config"
	"github.com/gavilanbe/woffuk-cli/internal/woffu"
)

var signCmd = &cobra.Command{
	Use:   "sign",
	Short: "Clock in/out on Woffu (works locally and in CI)",
	RunE: func(cmd *cobra.Command, args []string) error {
		// Load config from file or env vars (CI fallback)
		cfg, password, err := config.LoadOrEnv()
		if err != nil {
			return err
		}

		client := woffu.NewWoffuClient(cfg.WoffuURL)
		companyClient := woffu.NewCompanyClient(cfg.WoffuCompanyURL)

		fmt.Println("Authenticating...")
		token, err := woffu.Authenticate(client, companyClient, cfg.WoffuEmail, password)
		if err != nil {
			return fmt.Errorf("auth failed: %w", err)
		}

		fmt.Println("Checking calendar...")
		info, err := woffu.GetSignInfo(companyClient, token)
		if err != nil {
			return fmt.Errorf("get sign info: %w", err)
		}

		if !info.ShouldSign {
			fmt.Println("No need to sign today.")
			return nil
		}

		lat, lon := cfg.Latitude, cfg.Longitude
		if info.IsTelework {
			lat, lon = cfg.HomeLatitude, cfg.HomeLongitude
			fmt.Printf("Telework day — signing with home coordinates (%.4f, %.4f)\n", lat, lon)
		} else {
			fmt.Printf("Office day — signing with office coordinates (%.4f, %.4f)\n", lat, lon)
		}

		signID, err := woffu.DoSign(companyClient, token, lat, lon)
		if err != nil {
			return fmt.Errorf("sign failed: %w", err)
		}

		fmt.Printf("Signed successfully! Event ID: %s\n", signID)
		return nil
	},
}
