package record

import (
	"fmt"
	"os"
	"os/exec"

	defaults "github.com/compspec/compat-lib/pkg/fs"
	"github.com/compspec/compat-lib/pkg/logger"
	"github.com/google/shlex"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// Generic update type to be parsed later
type Update struct {
	Message string
}

type RecordFS struct {
	Server     *fuse.Server
	MountPoint string

	// Output file, if defined, to save events
	Outfile string
}

// Cleanup removes the mountpoint directory
func (rfs *RecordFS) Cleanup() {

	// Clean up mount point directory
	fmt.Printf("Cleaning up %s...\n", rfs.MountPoint)
	os.RemoveAll(rfs.MountPoint)
	if logger.Outfile != "" {
		fmt.Printf("Output file written to %s\n", logger.Outfile)
		os.Chmod(logger.Outfile, 0644)
	}
}

// NewRecordFS returns a new wrapper to a fuse.Server
// We mount a fusefs to a temporary directory
// The server returned (if not nil) needs to be
// correctly handled - see how it is used here in the library
// If recorder is true, we instantiate a recording base
func NewRecordFS(
	mountPath string,
	recordFile string,
	readOnly bool,
) (*RecordFS, error) {

	// Create a Compat Filesystem with defaults
	rfs := RecordFS{Outfile: logger.Outfile}

	// Set the global log file in case we are recording events
	logger.SetOutfile(recordFile)

	// TODO keep track of cpu and memory profiles
	if mountPath == "" {
		mountPoint, err := os.MkdirTemp("", "recordfs")
		if err != nil {
			return nil, err
		}
		mountPath = mountPoint
	}

	// One more check if directory doesn't exist
	_, err := os.Stat(mountPath)
	if err != nil && os.IsNotExist(err) {
		err := os.Mkdir(mountPath, 0755)
		if err != nil {
			return nil, err
		}
	}
	// Create generic updates channel
	updates := make(chan Update)

	fmt.Printf("Mount directory %s\n", mountPath)
	rfs.MountPoint = mountPath

	// Mount the content of the rootFS (originalFS) at the mount point
	// Pass in a channel to receive updates from
	err = rfs.InitLoopbackRoot(
		defaults.OriginalFS,
		mountPath,
		updates,
		readOnly,
	)
	if err != nil {
		return nil, err
	}
	return &rfs, nil
}

// RunComand to the fuse filesystem with chroot
func (rfs *RecordFS) RunCommand(command string) error {

	// returns list of strings
	call, err := shlex.Split(command)
	if err != nil {
		return err
	}

	command, args := call[0], call[1:]

	// Get current working directory to return to
	cwd, err := os.Getwd()
	if err != nil {
		return err
	}

	// Setup command, using standard outputs
	cmd := exec.Command(command, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	cmd.Dir = cwd

	err = cmd.Run()
	if err != nil {
		return err
	}
	return err
}
