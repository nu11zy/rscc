package sshd

import (
	"agent/internal/logger"
	"fmt"
	"io"
	"os/exec"

	"github.com/google/shlex"
	"golang.org/x/crypto/ssh"
)

func handleExec(channel ssh.Channel, req *ssh.Request) error {
	lg := logger.GetLogger()

	var command struct {
		Cmd string
	}

	// get cmd to exec
	if err := ssh.Unmarshal(req.Payload, &command); err != nil {
		req.Reply(false, nil)
		return err
	}
	req.Reply(true, nil)

	lg.Info("Exec command: %s", command.Cmd)

	// get arguments
	splitted, err := shlex.Split(command.Cmd)
	if err != nil {
		return fmt.Errorf("split arguments: %v", err)
	}
	execCmd := splitted[0]
	execArgs := []string{}
	if len(splitted) > 1 {
		execArgs = splitted[1:]
	}

	cmd := exec.Command(execCmd, execArgs...)
	// get stdout for command
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		_, _ = channel.Write([]byte(fmt.Sprintf("get stdout: %v\n", err)))
		return fmt.Errorf("get stdout: %v", err)
	}
	defer stdout.Close()
	// set stderr as stdout
	cmd.Stderr = cmd.Stdout
	// get stdin for command
	stdin, err := cmd.StdinPipe()
	if err != nil {
		_, _ = channel.Write([]byte(fmt.Sprintf("get stdin: %v\n", err)))
		return fmt.Errorf("get stdin: %v", err)
	}
	defer stdin.Close()

	go io.Copy(stdin, channel)
	go io.Copy(channel, stdout)

	// execute
	if err := cmd.Run(); err != nil {
		_, _ = channel.Write([]byte(fmt.Sprintf("%v\n", err)))
		return err
	}
	return nil
}
