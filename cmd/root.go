package cmd

import (
	tea "charm.land/bubbletea/v2"
	"github.com/cheetahbyte/apex/internal/agent"
	"github.com/cheetahbyte/apex/internal/config"
	llmproviders "github.com/cheetahbyte/apex/internal/llm/providers"
	"github.com/cheetahbyte/apex/internal/llm/toolclient"
	"github.com/cheetahbyte/apex/internal/tools/builtin"
	"github.com/cheetahbyte/apex/internal/tui"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "apex",
	Short: "Apex is a terminal coding agent",
	RunE: func(cmd *cobra.Command, args []string) error {
		cfg := config.Default()
		base, provider, err := newLLMClient(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		client := toolclient.New(base, toolclient.ModeFromString(string(cfg.ToolMode)))
		registry := builtin.NewRegistry()
		apexAgent := agent.NewWithContextWindow(client, registry, llmproviders.ContextWindowForModel(cfg.Model))
		_, err = tea.NewProgram(tui.New(apexAgent, tui.RuntimeInfo{Provider: provider.ID, Model: cfg.Model})).Run()
		return err
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
