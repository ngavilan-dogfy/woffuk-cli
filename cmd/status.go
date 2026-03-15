package cmd

import (
	"fmt"

	"github.com/spf13/cobra"

	"github.com/ngavilan-dogfy/woffuk-cli/internal/config"
	"github.com/ngavilan-dogfy/woffuk-cli/internal/woffu"
)

var statusCmd = &cobra.Command{
	Use:   "status",
	Short: "Show today's signing status",
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

		info, err := woffu.GetSignInfo(companyClient, token)
		if err != nil {
			return fmt.Errorf("get sign info: %w", err)
		}

		fmt.Printf("Date:        %s\n", info.Date)
		fmt.Printf("Should sign: %s\n", boolToYesNo(info.ShouldSign))
		fmt.Printf("Telework:    %s\n", boolToYesNo(info.IsTelework))

		if len(info.NextEvents) > 0 {
			fmt.Println("\nNext events:")
			for _, e := range info.NextEvents {
				names := ""
				if len(e.Names) > 0 {
					names = " — " + e.Names[0]
				}
				fmt.Printf("  %s%s\n", e.Date, names)
			}
		}

		return nil
	},
}

func boolToYesNo(b bool) string {
	if b {
		return "yes"
	}
	return "no"
}
