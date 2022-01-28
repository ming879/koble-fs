package startup

import (
	"fmt"
	"os"
	"os/exec"

	log "github.com/sirupsen/logrus"
)

func runStartup(path string) error {
	_, err := os.Stat(path)
	if err != nil && !os.IsNotExist(err) {
		return err
	} else if os.IsNotExist(err) {
		return nil
	}
	cmd := exec.Cmd{
		Path:   "/bin/bash",
		Args:   []string{"-c", path},
		Stdout: os.Stdout,
		Stderr: os.Stderr,
	}
	return cmd.Run()
}

func StartPhaseTwo() error {
	conf, err := LoadConf()
	if err != nil {
		return err
	}
	err = runStartup("/hostlab/shared.startup")
	if err != nil {
		log.Error("error running shared startup script: %v", err)
	}
	err = runStartup(fmt.Sprintf("/hostlab/%s.startup", conf.HostName))
	if err != nil {
		log.Error("error running machine startup script: %v", err)
	}
	fmt.Println("driver is", conf.Driver)
	if conf.Driver == "UML" {
		return os.WriteFile("/run/uml-guest/machine.ready", []byte(""), 0644)
	}
	return nil
}
