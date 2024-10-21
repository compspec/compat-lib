package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"path/filepath"
	"strings"
	"syscall"

	"github.com/compspec/compat-lib/pkg/fs"
	"github.com/compspec/compat-lib/pkg/generate"
)

func main() {
	fmt.Println("⭐️ Compatibility Filesystem (fs-gen)")
	mountPoint := flag.String("mount-path", "", "Mount path (for control from calling process)")

	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		log.Fatalf("You must provide a command (with optional arguments) to run.")
	}
	mountPath := *mountPoint

	// Get the full path of the command
	path := args[0]
	path, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("Error getting full path: %x", err)

	}

	// This is where we should look them up in some cache
	fmt.Printf("Preparing to find shared libraries needed for %s\n", args)
	_, err = generate.FindSharedLibs(path)
	if err != nil {
		log.Panicf("Error finding shared libraries for %s: %x", path, err)
	}

	// Generate the fusefs server
	compatFS, err := fs.NewCompatFS(mountPath)
	if err != nil {
		log.Panicf("Cannot generate fuse server: %x", err)
	}
	fmt.Println("Mounted!")

	// Removes mount point directory when done
	defer compatFS.Cleanup()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		compatFS.Server.Unmount()
	}()

	// Execute the command with proot
	proot := []string{"proot", "-S", compatFS.MountPoint, "-0"}
	args = append(proot, args...)
	call := strings.Join(args, " ")
	fmt.Println(call)
	err = compatFS.RunCommand(call)
	if err != nil {
		log.Panicf("Error running command: %s", err)
	}
	defer compatFS.Server.Unmount()
	compatFS.Server.Wait()
}
