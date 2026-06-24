package cmd

import (
	tea "charm.land/bubbletea/v2"
	"github.com/cheetahbyte/apex/internal/agent"
	"github.com/cheetahbyte/apex/internal/config"
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
		base, err := newLLMClient(cmd.Context(), cfg)
		if err != nil {
			return err
		}
		client := toolclient.New(base, toolclient.ModeFromString(string(cfg.ToolMode)))
		registry := builtin.NewRegistry()
		apexAgent := agent.New(client, registry)
		_, err = tea.NewProgram(tui.New(apexAgent)).Run()
		return err
	},
}

func Execute() {
	cobra.CheckErr(rootCmd.Execute())
}
