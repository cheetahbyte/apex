package cmd

import (
	tea "charm.land/bubbletea/v2"
	"github.com/cheetahbyte/apex/internal/config"
	"github.com/cheetahbyte/apex/internal/llm"
	"github.com/cheetahbyte/apex/internal/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "apex",
	Short: "Apex is a terminal coding agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Default()
		client := llm.NewOpenAIClient(cfg.Model, cfg.BaseURL, cfg.APIKey)
		_, err := tea.NewProgram(tui.New(client)).Run()
		return err
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
