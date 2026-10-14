package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
	"github.com/wowsims/classic/sim/ruleset"
)

var rulesetID string

var rootCmd = &cobra.Command{
	Use:   "wowsimcli",
	Short: "wowsims command line tool",
	Long:  "wowsims command line tool",
	PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
		return ruleset.Activate(rulesetID)
	},
}

func init() {
	rootCmd.PersistentFlags().StringVar(&rulesetID, "ruleset", ruleset.Default, fmt.Sprintf("game version to simulate (available: %v)", ruleset.IDs()))
}

func Execute(version string) {
	rootCmd.AddCommand(newVersionCommand(version))
	rootCmd.AddCommand(simCmd)
	rootCmd.AddCommand(bulkCmd)
	rootCmd.AddCommand(decodeLinkCmd)

	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}
