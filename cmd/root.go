package main

import (
	"github.com/b177y/koble-fs/pkg/startup"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:           "kstart",
	Short:         "kstart is used by koble guests for managing startup",
	SilenceUsage:  true,
	SilenceErrors: true,
}

var ph1 = &cobra.Command{
	Use:   "phase1",
	Short: "start phase1 of koble startup",
	RunE: func(cmd *cobra.Command, args []string) error {
		return startup.StartPhaseOne()
	},
}

var ph2 = &cobra.Command{
	Use:   "phase2",
	Short: "start phase2 of koble startup",
}

var shutdown = &cobra.Command{
	Use:   "shutdown",
	Short: "start of koble shutdown",
}

func init() {
	rootCmd.AddCommand(ph1)
	rootCmd.AddCommand(ph2)
}
