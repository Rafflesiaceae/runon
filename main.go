package main

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"runon/shell_quote"
)

const helpMsg = `usage: <host> cmd ...`

// Config stores per-project hooks and configuration
type Config struct {
	Ignore   []string `yaml:"ignore,omitempty"`
	OnChange []string `yaml:"on change,omitempty"`
}

// @TODO pass args that copy produced files back again

// @TODO CLI / flags/args for passing -S - also check if socket exists and offer an option accordingly if it doesnt

// @TODO run daemon in named-tmux instance - possibly restart it if run more then once
// @TODO when on-changed commands fail, re-run them next time even if no files changed?

// @TODO cleanup on panic

// @TODO CLI / flag to reset/delete the remote project directory
// @TODO add clean command in case we accidentally transmitted build artifacts we later ignored // command for cleaning up all paths that are now ignored

// @TODO info-level logging gives timings for profiling (when rsync is finished etc)
// @TODO support multiple runon files you can pick from, autocomplete according to a pattern, e.g.: `.runon.windows.yaml`
// @TODO CLI / flag to reset/delete the remote project directory

var master *exec.Cmd

var (
	remoteShellCmd = "bash -lc"
)

func init() {
	log.SetOutput(os.Stderr)
	log.SetLevel(log.InfoLevel)
}

func parseYml(configPath string) (config *Config, errRet error) {
	var err error

	// check if file exists
	_, err = os.Stat(configPath)

	if err != nil {
		return nil, err // file doesnt exist or other error
	}

	// parse config
	config = &Config{}

	rawConfig, err := ioutil.ReadFile(configPath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(rawConfig, &config)
	if err != nil {
		return nil, err
	}

	return
}

func sshCommand(socketPath string, host string, stdoutToStderr bool, remoteProjectPath string, remoteCmd string) error {
	cmdArgs := []string{
		"-S",
		socketPath,
		host,
	}

	isInteractiveTerminal := false
	if remoteCmd == "" {
		isInteractiveTerminal = true
	}

	if !isInteractiveTerminal {
		cmdArgs = append(cmdArgs,
			"--",
			fmt.Sprintf(
				"%s %s", remoteShellCmd, shell_quote.ShBackslashQuote(
					fmt.Sprintf("cd %s && %s", remoteProjectPath, remoteCmd),
				),
			),
		)
	}

	cmd := exec.Command("ssh", cmdArgs...)
	// @TODO run this command in a shell this probably solves the $TERM issues

	log.Debugf("%v", cmd.Args)

	cmd.Stdin = os.Stdin
	if stdoutToStderr {
		cmd.Stdout = os.Stderr
	} else {
		cmd.Stdout = os.Stdout

	}
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	if !isInteractiveTerminal && err != nil {
		log.Errorf("remote command failed: \"%v\"\n", remoteCmd)
		return err
	}

	return nil
}

// RunMasterOnly Runs a master daemon for a given host without running commands
func RunMasterOnly(host string) {
	socketPath := AssembleDefaultSocketPath(host)
	master := NewControlMaster(socketPath, host)
	if master != nil {
		defer master.Cleanup()
		log.Infof("control-master starting up, listening on: \"%s\"\n", master.SocketPath)

		// wait for the master process to close
		if err := master.Cmd.Wait(); err != nil {
			log.Error(err)
			os.Exit(255)
		}
		os.Exit(0)
	} else {
		log.Errorf("master-control already listening on \"%s\"", socketPath)
	}
}

func assemblePaths() (remoteProjectPath string, projectPathHashVal string) {
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	hostname, err := os.Hostname()
	if err != nil {
		panic(err)
	}

	projectPathHashVal = hostname + ":" + cwd
	log.Debugf("project path hashval: %s", projectPathHashVal)

	projectPathHash := fmt.Sprintf("%04X", sha256.Sum256([]byte(projectPathHashVal)))[:16] // @NOTE breaking any security guarantee of sha256
	remoteProjectPath = fmt.Sprintf("~/.runon/%s", projectPathHash)

	return remoteProjectPath, projectPathHashVal
}

// Run executes a list of commands on a given host
func Run(host string, cmdArgs []string) {
	var err error

	remoteProjectPath, projectPathHashVal := assemblePaths()

	log.Debugf("remote project path: %s", remoteProjectPath)

	// parsePotential config
	config, err := parseYml("./.runon.yml")
	if err != nil && !os.IsNotExist(err) { // in case we found a file and an error occured during parsing
		panic(err)
	}

	socketPath := AssembleDefaultSocketPath(host)
	master := NewControlMaster(socketPath, host)
	defer master.Cleanup()

	var rsyncStdout string
	{ // rsync
		ignoreList := []string{}
		if config != nil && config.Ignore != nil {
			ignoreList = config.Ignore
		}

		// assemble filterList from ignoreList
		filterList := []string{}
		for _, v := range ignoreList {
			filterList = append(filterList, fmt.Sprintf("--exclude=%s", v))
		}

		// call rsync
		rsyncArgs := append(append(
			[]string{
				"-ar",
				"-i", // print status to stdout
			},
			filterList...,
		),
			[]string{
				"-e", fmt.Sprintf("ssh -o ControlPath=%s", socketPath), // use ssh
				".", // source
				fmt.Sprintf("%s:%s", host, remoteProjectPath), // target
			}...,
		)

		log.Debugf("rsync arguments: %v", rsyncArgs)

		rsyncStdout, _, err = CheckExec("rsync", rsyncArgs...)
		log.Debugf("rsync output: \"%v\"", rsyncStdout)
		if err != nil {
			panic(err)
		}

		if strings.HasPrefix(rsyncStdout, "cd+++++++++ ./") { // remoteProjectPath was created for the first time
			log.Info("first time creating remote-project")
			err := sshCommand(socketPath, host, true, remoteProjectPath, fmt.Sprintf("echo \"$(pwd)\t%s\" >> ../projects", projectPathHashVal))
			if err != nil {
				log.Fatal(err)
			}

		}

	}

	// possibly run the on-changed commands on host
	if len(rsyncStdout) > 0 { // some files were changed
		// assemble onChangeList
		onChangeList := []string{}
		if config != nil && config.Ignore != nil {
			onChangeList = config.OnChange
		}

		// run commands on-change
		for _, v := range onChangeList {
			log.Infof("running onChange command: \"%v\"\n", v)

			err := sshCommand(socketPath, host, true, remoteProjectPath, v)
			if err != nil {
				log.Fatal(err)
			}
		}
	}

	// run passed command
	if len(cmdArgs) > 0 {
		log.Infof("running command: \"%v\"\n", cmdArgs)
		err = sshCommand(socketPath, host, false, remoteProjectPath, strings.Join(cmdArgs, " "))
	} else {
		log.Infoln("starting interactive terminal")
		sshCommand(socketPath, host, false, remoteProjectPath, "") // drop down to shell
	}
	if err != nil {
		log.Error(err)
	}
}

// InitConfig initializes a bare config in the cwd
func InitConfig() {
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
    - .svn
    - node_modules
    - build

# on change:`), 0644)
	if err != nil {
		log.Fatal(err)
	}

}

// Clean cleans the current project that's stored on a given host
func Clean(host string) {
	remoteProjectPath, projectPathHashVal := assemblePaths()
	if projectPathHashVal == "" {
		//
	}

	socketPath := AssembleDefaultSocketPath(host)
	master := NewControlMaster(socketPath, host)
	defer master.Cleanup()

	log.Infof("cleaning project (%s)", remoteProjectPath)
	err := sshCommand(socketPath, host, true, remoteProjectPath, fmt.Sprintf("cd .. && rm -rf %s", remoteProjectPath))
	if err != nil {
		log.Fatal(err)
	}
}
