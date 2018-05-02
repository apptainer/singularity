package runtime

import (
	"log"
	"os"
	"syscall"
)

func (engine *RuntimeEngine) PrestartProcess() error {
	/* seccomp setup goes here */
	return nil
}

func (engine *RuntimeEngine) StartProcess() error {
	//    if cconf.isInstance == C.uchar(0) {
	os.Setenv("PS1", "shell> ")
	os.Chdir("/")
	args := engine.OciConfig.RuntimeOciSpec.Process.Args
	err := syscall.Exec(args[0], args, os.Environ())
	if err != nil {
		log.Fatalf("exec %s failed: %s\n", args[0], err)
	}
	/*    }  else {
	          err := syscall.Exec("/bin/sleep", []string{"/bin/sleep", "60"}, os.Environ())
	          if err != nil {
	              log.Fatalln("exec failed:", err)
	          }
	      }
	*/
	return nil
}
