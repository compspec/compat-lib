package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	fs "github.com/compspec/compat-lib/pkg/fs/spindle"
	"github.com/compspec/compat-lib/pkg/generate"
	"github.com/compspec/compat-lib/pkg/utils"
)

func main() {
	fmt.Println("⭐️ Compatibility Filesystem (fs-gen)")
	mountPoint := flag.String("mount-path", "", "Mount path (for control from calling process)")
	readOnly := flag.Bool("read-only", true, "Read only mode (off by default)")

	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		log.Fatalf("You must provide a command (with optional arguments) to run.")
	}
	mountPath := *mountPoint

	// Get the full path of the command
	path := args[0]
	path, err := utils.FullPath(path)
	if err != nil {
		log.Fatalf("Error getting full path: %x", err)
	}
	// Ensure paths at 0 is fullpath
	args[0] = path

	// This is where we should look them up in some cache
	// This isn't currently used, but we would likely have a view of the system
	// that can do some kind of pre-cache thing...
	fmt.Printf("Preparing to find shared libraries needed for %s\n", args)
	_, err = generate.FindSharedLibs(path)
	if err != nil {
		log.Panicf("Error finding shared libraries for %s: %x", path, err)
	}

	// Generate the fusefs server
	sfs, err := fs.NewSpindleFS(mountPath, "", *readOnly)
	if err != nil {
		log.Panicf("Cannot generate fuse server: %x", err)
	}
	fmt.Println("Mounted!")

	// Removes mount point directory when done
	defer sfs.Cleanup()

	here, err := os.Getwd()
	if err != nil {
		log.Panicf("Cannot get current working directory: %x", err)
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		sfs.Server.Unmount()
	}()

	// Execute the command with proot, w is for pwd/cwd
	proot := []string{"proot", "-S", sfs.MountPoint, "--kill-on-exit", "-w", here}
	args = append(proot, args...)
	call := strings.Join(args, " ")
	fmt.Println(call)
	err = sfs.RunCommand(call)
	if err != nil {
		log.Panicf("Error running command: %s", err)
	}

	// Unlike compat, explicitly close after command is done running
	fmt.Println("Command is done running")
	sfs.Server.Unmount()
}
