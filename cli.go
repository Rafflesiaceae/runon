package main

import (
	"fmt"
	"os"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	flagDebug bool
)

var rootCmd = &cobra.Command{
	Use:   "runon [host] [args]...",
	Short: "runon syncs workspaces of projects, optionally runs commands in them if changes needed syncing and then runs passed commands",
	Args:  cobra.MinimumNArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		persistentFlagCommonalities(cmd)

		host := args[0]
		runArgs := args[1:]

		Run(host, runArgs)
	},
}

var masterControlCommand = &cobra.Command{
	Use:   "master [host]",
	Short: "Run MasterControl in daemon mode",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		persistentFlagCommonalities(cmd)

		host := args[0]
		RunMasterOnly(host)
	},
}

var initCommand = &cobra.Command{
	Use:   "init",
	Short: "Init bare .runon.yml file",
	Run: func(cmd *cobra.Command, args []string) {
		persistentFlagCommonalities(cmd)

		InitConfig()
	},
}

var cleanCommand = &cobra.Command{
	Use:   "clean [host]",
	Short: "Clean remote dir",
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		persistentFlagCommonalities(cmd)

		host := args[0]
		Clean(host)
	},
}

func persistentFlagCommonalities(cmd *cobra.Command) {
	if flagDebug {
		log.SetLevel(log.DebugLevel)
		log.Info("enabling debug-mode")
	}
}

func init() {
	rootCmd.AddCommand(masterControlCommand)
	rootCmd.AddCommand(cleanCommand)
	rootCmd.AddCommand(initCommand)

	rootCmd.PersistentFlags().BoolVar(&flagDebug, "debug", false, "Enable debug output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
