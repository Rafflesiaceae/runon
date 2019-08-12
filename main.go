package main

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"

	"runon/shell_quote"
)

const helpMsg = `usage: <host> cmd ...`

type Config struct {
	Ignore   []string `yaml:"ignore,omitempty"`
	OnChange []string `yaml:"on change,omitempty"`
}

// @TODO cleanup on panic
// @TODO CLI / flags/args for passing -S - also check if socket exists and offer an option accordingly if it doesnt
// @TODO CLI / init writes down a bare .runon.yml with commented out stuff
// @TODO pass args that copy produced files back again
// @TODO run daemon in named-tmux instance - possibly restart it if run more then once
// @TODO when on-changed commands fail, re-run them next time even if no files changed?
// @TODO add clean command in case we accidentally transmitted build artifacts we later ignored // command for cleaning up all paths that are now ignored
// @TODO info-level logging gives timings for profiling (when rsync is finished etc)
// @TODO support multiple runon files you can pick from, autocomplete according to a pattern, e.g.: `.runon.windows.yaml`

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

func Run(host string, cmdArgs []string) {
	var err error

	// create unique hash out of current working dir
	cwd, err := os.Getwd()
	if err != nil {
		panic(err)
	}

	projectPathHash := fmt.Sprintf("%04X", sha256.Sum256([]byte(cwd)))[:16] // @XXX breaking any security guarantee of sha256
	remoteProjectPath := fmt.Sprintf("~/.runon/%s", projectPathHash)

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
		rsyncStdout, _, err = CheckExec("rsync", append(append(
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
		)...)
		if err != nil {
			panic(err)
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
				panic(err)
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
