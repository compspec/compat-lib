package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	fs "github.com/compspec/compat-lib/pkg/fs/record"
	"github.com/compspec/compat-lib/pkg/logger"
	"github.com/compspec/compat-lib/pkg/utils"
)

func main() {
	fmt.Println("⭐️ Filesystem Recorder (fs-record)")

	mountPoint := flag.String("mount-path", "", "Mount path (for control from calling process)")
	outfile := flag.String("out", "", "Output file to write events")
	outdir := flag.String("out-dir", "", "Output directory to write events")
	readOnly := flag.Bool("read-only", true, "Read only mode (off by default)")

	flag.Parse()
	args := flag.Args()
	if len(args) == 0 {
		log.Fatal("You must provide a command (with optional arguments) to run.")
	}
	mountPath := *mountPoint

	// Get the full path of the command
	path := args[0]
	path, err := utils.FullPath(path)
	if err != nil {
		fmt.Println(err)
		log.Fatal("error getting full path")
	}
	args[0] = path

	// We require a recording file for the recorder
	if *outfile == "" {
		*outfile = logger.GetEventFile(*outdir)
	}
	// Generate the fusefs server
	rfs, err := fs.NewRecordFS(mountPath, *outfile, *readOnly)
	if err != nil {
		fmt.Println(err)
		log.Panic("cannot generate fuse server")
	}
	fmt.Println("Mounted!")

	// Removes mount point directory when done
	// Also fixes permission of file
	defer rfs.Cleanup()

	c := make(chan os.Signal)
	signal.Notify(c, os.Interrupt, syscall.SIGTERM)
	go func() {
		<-c
		rfs.Server.Unmount()
	}()

	// Execute the command with proot
	proot := []string{"proot", "-S", rfs.MountPoint, "-0"}
	args = append(proot, args...)
	call := strings.Join(args, " ")
	fmt.Println(call)
	err = rfs.RunCommand(call)

	// Record the end of command event.
	logger.LogEvent("Complete", logger.Outfile)
	if err != nil {
		fmt.Println(err)
		log.Panic("error running command")
	}
	// Unlike compat, explicitly close after command is done running
	fmt.Println("Command is done running")
	rfs.Server.Unmount()
}
