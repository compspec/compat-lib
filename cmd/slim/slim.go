package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	fs "github.com/compspec/compat-lib/pkg/fs/slim"
	"github.com/compspec/compat-lib/pkg/utils"
)

func main() {
	fmt.Println("🥕 Container Slimmer (slim)")

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
		fmt.Println(err)
		log.Fatalf("Error getting full path")
	}

	// Generate the fusefs server
	sfs, err := fs.NewSlimFS(mountPath, *outfile, *readOnly)
	if err != nil {
		fmt.Println(err)
		log.Panicf("Cannot generate fuse server")
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

	// Removes mount point directory
	// Here we keep the cache, we cannot remove
	defer sfs.Cleanup(*keepCache)

	// Working directory to run command from
	here := *workdir
	if here == "" {
		here, err = os.Getwd()
		if err != nil {
			fmt.Println(err)
			log.Panicf("Cannot get current working directory")
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
		fmt.Println(err)
		log.Panicf("Cannot find proot executable")
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
		fmt.Println(err)
		log.Panicf("Error running command")
	}

	// Unlike compat, explicitly close after command is done running
	fmt.Println("Command is done running")
	if *wait {
		sfs.Server.Wait()
	}
	sfs.Server.Unmount()
}
