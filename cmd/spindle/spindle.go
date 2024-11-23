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
	fmt.Println("🧵 Filesystem Cache (spindle)")

	// Note that most of the cache optimization happens depending on where you do the mount (and create the cache)
	// It's using this cache that will bypass calls to the other filesystem (e.g., NFS) at least I think :)
	mountPoint := flag.String("mount-path", "", "Mount path for fuse root and cache (created in /tmp/spindleXXXXX if does not exist)")
	workdir := flag.String("workdir", "", "Working directory (defaults to pwd)")
	wait := flag.Bool("wait", false, "Wait (and do not unmount) at the end (off by default)")
	readOnly := flag.Bool("read-only", true, "Read only mode (on by default, as the layer to intercept does not need write)")
	verbose := flag.Bool("v", false, "Run proot in verbose mode (off by default)")
	outfile := flag.String("out", "", "Output file to write events (unset will not write anything anywhere)")
	keepCache := flag.Bool("keep", false, "Do not cleanup the cache")

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

	// This is where we should look them up in some cache
	// This isn't currently used, but we would likely have a view of the system
	// that can do some kind of pre-cache thing...
	fmt.Printf("Preparing to find shared libraries needed for %s\n", args)
	_, err = generate.FindSharedLibs(path)
	if err != nil {
		log.Panicf("Error finding shared libraries for %s: %x", path, err)
	}

	// Generate the fusefs server
	sfs, err := fs.NewSpindleFS(mountPath, *outfile, *readOnly)
	if err != nil {
		log.Panicf("Cannot generate fuse server: %x", err)
	}
	fmt.Println("Mounted!")
	fmt.Printf("   ReadOnly: %t\n", *readOnly)
	fmt.Printf("    Verbose: %t\n", *verbose)
	fmt.Printf("    Cleanup: %t\n", *keepCache)
	fmt.Printf("      Cache: %s\n", sfs.CacheFS())
	fmt.Printf("       Root: %s\n", sfs.RootFS())

	// Scope the application to the space of the fuse mount
	// Ensure paths at 0 is fullpath
	mountedPath := sfs.MountedPath(path)
	args[0] = mountedPath

	// Removes mount point directo
	defer sfs.Cleanup(*keepCache)

	// Working directory to run command from
	here := *workdir
	if here == "" {
		here, err = os.Getwd()
		if err != nil {
			log.Panicf("Cannot get current working directory: %x", err)
		}
	}

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		sfs.Server.Unmount()
	}()

	// Ensure we have proot
	proot, err := utils.FullPath("proot")
	if err != nil {
		log.Panicf("Cannot find proot executable: %x", err)
	}

	// Execute the command in the context of the fuse mount root
	command := []string{proot, "-R", sfs.RootFS(), "--kill-on-exit", "-w", here}
	if *verbose {
		command = []string{proot, "-v", "1", "-R", sfs.RootFS(), "--kill-on-exit", "-w", here}
	}
	args = append(command, args...)
	call := strings.Join(args, " ")
	fmt.Println(call)
	err = sfs.RunCommand(call, here)
	if err != nil {
		log.Panicf("Error running command: %s", err)
	}

	// Unlike compat, explicitly close after command is done running
	fmt.Println("Command is done running")
	if *wait {
		sfs.Server.Wait()
	}
	sfs.Server.Unmount()
}
