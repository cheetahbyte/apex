package cmd

import (
	"fmt"
	"strings"

	"github.com/cheetahbyte/apex/internal/config"
	llmproviders "github.com/cheetahbyte/apex/internal/llm/providers"
	"github.com/spf13/cobra"
)

var modelsCmd = &cobra.Command{
	Use:   "models [provider]",
	Short: "View all available models grouped by provider",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		manager, err := newAuthManager()
		if err != nil {
			return err
		}

		if len(args) == 1 {
			cfg := config.Default()
			cfg.Provider = args[0]
			provider, err := llmproviders.Resolve(cfg)
			if err != nil {
				return err
			}
			if !llmproviders.IsConfigured(cmd.Context(), provider, cfg, manager) {
				return fmt.Errorf("provider %q is not configured; run apex auth login %s", provider.ID, provider.ID)
			}
			models, err := llmproviders.ListModels(cmd.Context(), provider, cfg, manager)
			if err != nil {
				if llmproviders.IsModelListUnsupported(err) {
					printProviderUnsupportedModels(cmd, provider)
					return nil
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %v\n", err)
			}
			printProviderModels(cmd, provider, models)
			return nil
		}

		printed := false
		cfg := config.Config{}
		for _, provider := range llmproviders.All() {
			if !llmproviders.IsConfigured(cmd.Context(), provider, cfg, manager) {
				continue
			}
			models, err := llmproviders.ListModels(cmd.Context(), provider, cfg, manager)
			if err != nil {
				if llmproviders.IsModelListUnsupported(err) {
					if printed {
						cmd.Println()
					}
					printProviderUnsupportedModels(cmd, provider)
					printed = true
					continue
				}
				fmt.Fprintf(cmd.ErrOrStderr(), "warning: %s: %v\n", provider.ID, err)
			}
			if printed {
				cmd.Println()
			}
			printProviderModels(cmd, provider, models)
			printed = true
		}
		if !printed {
			cmd.Println("No providers configured. Run `apex auth login <provider>`.")
		}
		return nil
	},
}

func printProviderModels(cmd *cobra.Command, provider llmproviders.Provider, models []llmproviders.ModelSpec) {
	cmd.Println(provider.ID)
	for _, model := range models {
		name := model.DisplayName
		if strings.TrimSpace(name) == "" {
			name = model.ID
		}
		cmd.Printf("  %s\n", name)
	}
	if len(models) == 0 {
		cmd.Println("  no models found")
	}
}

func printProviderUnsupportedModels(cmd *cobra.Command, provider llmproviders.Provider) {
	cmd.Println(provider.ID)
	cmd.Println("  model listing not supported")
}

func init() {
	rootCmd.AddCommand(modelsCmd)
}
