package main

import (
	"crypto/sha256"
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	pathlib "path"
	"strings"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	"gopkg.in/yaml.v2"

	"runon/shell_quote"
)

const helpMsg = `usage: <host> cmd ...`

// Config stores per-project hooks and configuration
type Config struct {
	ProjectHash string   `yaml:"project-hash,omitempty"`
	Ignore      []string `yaml:"ignore,omitempty"`
	OnChange    []string `yaml:"on change,omitempty"`
}

// @TODO CLI / flags/args for passing -S - also check if socket exists and offer an option accordingly if it doesnt

// @TODO when on-changed commands fail, re-run them next time even if no files changed?
// @TODO cleanup on panic

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

func assemblePaths(config *Config) (remoteProjectPath string, projectPathHashVal string) {
	var projectPathHash string
	if config.ProjectHash == "" {
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

		projectPathHash = fmt.Sprintf("%04X", sha256.Sum256([]byte(projectPathHashVal)))[:16] // @NOTE breaking any security guarantee of sha256
	} else {
		projectPathHash = config.ProjectHash
	}

	remoteProjectPath = fmt.Sprintf("~/.runon/%s", projectPathHash)

	return remoteProjectPath, projectPathHashVal
}

// Run executes a list of commands on a given host
func Run(cmd *cobra.Command, host string, cmdArgs []string) {
	var err error

	copyBack := cmd.Flag("copy-back").Value.String()

	// parsePotential config
	config, err := parseYml("./.runon.yml")
	if err != nil && !os.IsNotExist(err) { // in case we found a file and an error occured during parsing
		panic(err)
	}

	remoteProjectPath, projectPathHashVal := assemblePaths(config)
	log.Debugf("remote project path: %s", remoteProjectPath)

	socketPath := AssembleDefaultSocketPath(host)
	master := NewControlMaster(socketPath, host)
	defer master.Cleanup()

	target := fmt.Sprintf("%s:%s", host, remoteProjectPath)
	var rsyncStdout string
	var rsync func(retry int)
	rsync = func(retry int) { // rsync
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
				".",    // source
				target, // target
			}...,
		)

		log.Debugf("rsync arguments: %v", rsyncArgs)

		var errCheck *ExecError
		rsyncStdout, _, errCheck = CheckExec("rsync", rsyncArgs...)
		log.Debugf("rsync output: \"%v\"", rsyncStdout)
		if errCheck != nil {
			// check if it's an error we can handle
			if retry < 1 &&
				strings.Contains(errCheck.stderr, "rsync:") &&
				strings.Contains(errCheck.stderr, "mkdir") &&
				strings.Contains(errCheck.stderr, "No such file or directory") {

				err = sshCommand(socketPath, host, true, "~", "mkdir -p \"$PWD/.runon\"")
				if err != nil {
					log.Fatal(err)
				}

				rsync(1)
				return
			} else {
				panic(errCheck)
			}
		}

		if strings.HasPrefix(rsyncStdout, "cd+++++++++ ./") { // remoteProjectPath was created for the first time
			log.Info("first time creating remote-project")
			err := sshCommand(socketPath, host, true, remoteProjectPath, fmt.Sprintf("echo \"$(pwd)\t%s\" >> ../projects", projectPathHashVal))
			if err != nil {
				log.Fatal(err)
			}

		}

	}
	rsync(0)

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

	// possibly copy files back to local
	if copyBack != "" {
		paths := strings.Split(copyBack, ";")
		absPaths := []string{}
		for _, path := range paths {
			absPaths = append(absPaths, pathlib.Join(target, path))
		}

		{ // rsync
			// call rsync
			rsyncArgs := append(append(
				[]string{
					"-ar",
					"-i",                                                   // print status to stdout
					"-e", fmt.Sprintf("ssh -o ControlPath=%s", socketPath), // use ssh
				},
				absPaths...,
			),
				[]string{
					pathlib.Join(".", "copyback-"+host) + "/", // source
				}...,
			)

			log.Debugf("rsync arguments: %v", rsyncArgs)

			rsyncStdout, _, err = CheckExec("rsync", rsyncArgs...)
			log.Debugf("rsync output: \"%v\"", rsyncStdout)
			if err != nil {
				panic(err)
			}

		}
	}
}

// InitConfig initializes a bare config in the cwd
func InitConfig() {
	var err error

	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	runonPath := pathlib.Join(cwd, ".runon.yml")
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
	var err error

	// parsePotential config
	config, err := parseYml("./.runon.yml")
	if err != nil && !os.IsNotExist(err) { // in case we found a file and an error occured during parsing
		panic(err)
	}

	remoteProjectPath, projectPathHashVal := assemblePaths(config)
	if projectPathHashVal == "" {
		//
	}

	socketPath := AssembleDefaultSocketPath(host)
	master := NewControlMaster(socketPath, host)
	defer master.Cleanup()

	log.Infof("cleaning project (%s)", remoteProjectPath)
	err = sshCommand(socketPath, host, true, remoteProjectPath, fmt.Sprintf("cd .. && rm -rf %s", remoteProjectPath))
	if err != nil {
		log.Fatal(err)
	}
}
