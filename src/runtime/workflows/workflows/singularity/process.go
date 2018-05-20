package runtime

import (
	"fmt"
	"os"
	"syscall"
)

// PrestartProcess runs pre-start tasks
func (engine *Engine) PrestartProcess() error {
	/* seccomp setup goes here */
	return nil
}

// StartProcess starts the process
func (engine *Engine) StartProcess() error {
	os.Setenv("PS1", "shell> ")
	os.Chdir("/")
	args := engine.OciConfig.RuntimeOciSpec.Process.Args
	err := syscall.Exec(args[0], args, os.Environ())
	if err != nil {
		return fmt.Errorf("exec %s failed: %s", args[0], err)
	}
	return nil
}
