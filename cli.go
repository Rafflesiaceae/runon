package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"path"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
)

var (
	flagDebug bool
)

var rootCmd = &cobra.Command{
	Use:   "runon [host] [args]...",
	Short: "runon syncs workspaces of projects, optionally runs commands in them if changes needed syncing and then runs passed commands",
	Long:  "",
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
	Long:  `All software has versions. This is Hugo's`,
	Args:  cobra.ExactArgs(1),
	Run: func(cmd *cobra.Command, args []string) {
		persistentFlagCommonalities(cmd)

		host := args[0]

		RunMasterOnly(host)
	},
}

var initCommand = &cobra.Command{
	Use:   "init",
	Short: "Initialize bare .runon.yml file in CWD",
	Run: func(cmd *cobra.Command, args []string) {
		persistentFlagCommonalities(cmd)

		var err error

		cwd, err := os.Getwd()
		if err != nil {
			log.Fatal(err)
		}

		runonPath := path.Join(cwd, ".runon.yml")
		if _, err = os.Stat(runonPath); err == nil {
			log.Errorf(".runon.yml file already exists: (%s)", runonPath)
			return
		}

		log.Infof("creating .runon.yml file (%s)", runonPath)
		err = ioutil.WriteFile(runonPath, []byte(`---
ignore:
    - .git
    - node_modules/**

on change:
    - ./build`), 0644)
		if err != nil {
			log.Fatal(err)
		}

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
	rootCmd.AddCommand(initCommand)

	rootCmd.PersistentFlags().BoolVar(&flagDebug, "debug", false, "Enable debug output")
}

func main() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Println(err)
		os.Exit(1)
	}
}
