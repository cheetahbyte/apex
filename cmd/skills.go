package cmd

import (
	"fmt"
	"sort"

	"charm.land/lipgloss/v2"
	"charm.land/lipgloss/v2/list"
	"github.com/cheetahbyte/apex/internal/skills"
	"github.com/spf13/cobra"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true)
	countStyle = lipgloss.NewStyle().
			Faint(true)
	itemStyle   = lipgloss.NewStyle()
	bulletStyle = lipgloss.NewStyle().
			Faint(true)
)

var skillsCmd = &cobra.Command{
	Use:   "skills",
	Short: "List available skills",
	RunE: func(cmd *cobra.Command, args []string) error {
		skillStore, err := skills.LoadDefault()
		if err != nil {
			return err
		}

		skillList, err := skillStore.List()
		if err != nil {
			return err
		}

		sort.Slice(skillList, func(i, j int) bool {
			return skillList[i].Name < skillList[j].Name
		})

		items := make([]any, 0, len(skillList))
		for _, skill := range skillList {
			items = append(items, skill.Name)
		}

		l := list.New(items...).
			Enumerator(func(_ list.Items, _ int) string {
				return bulletStyle.Render("• ")
			}).
			ItemStyle(itemStyle)

		cmd.Println(
			titleStyle.Render("Available skills"),
			countStyle.Render(fmt.Sprintf("%d", len(skillList))),
		)
		cmd.Println(l)

		return nil
	},
}

func init() {
	rootCmd.AddCommand(skillsCmd)
}
