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
)

func main() {
	fmt.Println("⭐️ Filesystem Recorder (fs-record)")

	mountPoint := flag.String("mount-path", "", "Mount path (for control from calling process)")
	outfile := flag.String("out", "", "Output file to write events")

	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		log.Fatalf("You must provide a command (with optional arguments) to run.")
	}
	mountPath := *mountPoint

	// Get the full path of the command
	path := args[0]
	_, err := filepath.Abs(path)
	if err != nil {
		log.Fatalf("Error getting full path: %x", err)

	}
	args[0] = path

	// We require a recording file for the recorder
	if *outfile == "" {
		*outfile = fs.GetEventFile()
	}
	// Generate the fusefs server
	compatFS, err := fs.NewCompatFS(mountPath, *outfile)
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