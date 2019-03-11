package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"os/exec"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"gopkg.in/yaml.v2"
)

const helpMsg = `usage: <host> cmd ...`

type Config struct {
	Ignore   []string `yaml:"ignore,omitempty"`
	OnChange []string `yaml:"on change,omitempty"`
}

// @TODO cleanup on panic
// @TODO put each project into its own dir instead of just into ~/.runon
// @TODO CLI / flags/args for passing -S - also check if socket exists and offer an option accordingly if it doesnt
// @TODO CLI / init writes down a bare .runon.yml with commented out stuff

var master *exec.Cmd

func forkSSHMaster(socketPath string, host string) (errorChan chan error) {
	errorChan = make(chan error, 1)

	go func() {
		// start master ssh command
		master = exec.Command(
			"ssh",
			"-M", // master
			"-S", socketPath,
			"-N", // don't execute a remoe command, just block
			host,
		)

		master.Stdin = os.Stdin
		master.Stdout = os.Stderr
		master.Stderr = os.Stderr

		err := master.Run()
		if err != nil {
			log.Errorln("failed trying to start the master ssh-daemon")
			errorChan <- err
		}

	}()

	// @XXX poll until socket is created >.<
	for {
		select {
		case err := <-errorChan:
			panic(err)
		default:
			_, err := os.Stat(socketPath)

			if err == nil { // file exists
				goto End
			} else if os.IsNotExist(err) { // file does not exist
				time.Sleep(40 * time.Millisecond)
			} else { // unknown error
				panic(err)
			}
		}
	}

End:
	return
}

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

func sshCommand(socketPath string, hostArg string, stdoutToStderr bool, args ...string) error {
	cmd := exec.Command("ssh",
		append(
			[]string{
				"-S",
				socketPath,
				hostArg,
			},
			args...,
		)...)

	cmd.Stdin = os.Stdin
	if stdoutToStderr {
		cmd.Stdout = os.Stderr
	} else {
		cmd.Stdout = os.Stdout

	}
	cmd.Stderr = os.Stderr
	err := cmd.Run()

	exitCode := GetExitCode(err)

	if err != nil {
		if exitCode == ExitCodeSSHUnknownCommand {
			log.Errorf("host does not know command: \"%v\"\n", args)
			log.Errorln("dropping down to shell...")

			cmd := exec.Command("ssh", "-S", socketPath, hostArg)
			cmd.Stdin = os.Stdin
			if stdoutToStderr {
				cmd.Stdout = os.Stderr
			} else {
				cmd.Stdout = os.Stdout

			}
			cmd.Stderr = os.Stderr
			err := cmd.Run()

			if err != nil {
				return err
			} else {
				return nil
			}
		}

		return err
	}

	return nil
}

func cleanup() {
}

func main() {

	var err error

	if len(os.Args) < 2 {
		println(helpMsg)
		os.Exit(-1)
	}

	// split args
	hostArg := os.Args[1]
	cmdArgs := os.Args[2:]

	// parsePotential config
	config, err := parseYml("./.runon.yml")
	if err != nil && !os.IsNotExist(err) { // in case we found a file and an error occured during parsing
		panic(err)
	}

	// make temp file
	socketPath, _, err := CaptureExec("mktemp", "-u", "/tmp/runon-ssh-socket.XXXXXX")
	if err != nil {
		panic(err)
	}

	socketPath = strings.TrimSpace(socketPath)

	_ = forkSSHMaster(socketPath, hostArg)

	defer func() { // clean up the socket
		_, err := os.Stat(socketPath)
		if err != nil {
			os.Remove(socketPath)
		}
	}()

	defer func() { // clean up master
		if master != nil {
			if master.Process != nil {
				err := master.Process.Kill()
				if err != nil {
					log.Error(err)
				}
			}
		}
	}()

	// rsync
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
	rsyncStdout, _, err := CaptureExec("rsync", append(append(
		[]string{
			"-ar",
			"-i", // print status to stdout
		},
		filterList...,
	),
		[]string{
			"-e", fmt.Sprintf("ssh -o ControlPath=%s", socketPath), // use ssh
			".",                                 // source
			fmt.Sprintf("%s:~/.runon", hostArg), // target // @TODO put each project in its own dir
		}...,
	)...)
	if err != nil {
		panic(err)
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
			err := sshCommand(socketPath, hostArg, true, "cd ~/.runon", "&&", v)
			if err != nil {
				panic(err)
			}
		}
	}

	// run passed command
	if len(cmdArgs) > 0 {
		log.Infof("running command: \"%v\"\n", cmdArgs)
		err = sshCommand(socketPath, hostArg, false, append([]string{"cd ~/.runon", "&&"}, cmdArgs...)...)
	} else {
		log.Infoln("starting interactive terminal")
		err = sshCommand(socketPath, hostArg, false, "cd ~/.runon")
	}
	if err != nil {
		log.Error(err)
	}
}
