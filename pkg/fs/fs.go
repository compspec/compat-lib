package fs

import (
	"fmt"
	"os"
	"os/exec"

	"github.com/google/shlex"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// Keep these constants for now
const (
	debug = false

	// mount with -o allowother
	other = false

	// Try to use "mount" syscall instead of fusermount
	directMount = false

	// Allow to fall back to fusermount (probably doesn't matter given directMount false)
	directMountStrict = false

	// original FS is for the loopback root
	originalFS = "/"
)

// Generic update type to be parsed later
type Update struct {
	Message string
}

type CompatFS struct {
	Server     *fuse.Server
	MountPoint string

	// Output file, if defined, to save events
	Outfile string
}

// Cleanup removes the mountpoint directory
func (c *CompatFS) Cleanup() {

	// Clean up mount point directory
	fmt.Printf("Cleaning up %s...\n", c.MountPoint)
	os.RemoveAll(c.MountPoint)
	if outfile != "" {
		fmt.Printf("Output file written to %s\n", outfile)
		os.Chmod(outfile, 0644)
	}
}

// NewCompatFS returns a new wrapper to a fuse.Server
// We mount a fusefs to a temporary directory
// The server returned (if not nil) needs to be
// correctly handled - see how it is used here in the library
// If recorder is true, we instantiate a recording base
func NewCompatFS(
	mountPath string,
	recordFile string,
) (*CompatFS, error) {

	// Create a Compat Filesystem with defaults
	compat := CompatFS{Outfile: outfile}

	// Set the global log file in case we are recording events
	outfile = recordFile

	// TODO keep track of cpu and memory profiles
	if mountPath == "" {
		mountPoint, err := os.MkdirTemp("", "compatlib")
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
	compat.MountPoint = mountPath

	// Mount the content of the rootFS (originalFS) at the mount point
	// Pass in a channel to receive updates from
	err = compat.InitLoopbackRoot(originalFS, mountPath, updates)
	if err != nil {
		return nil, err
	}
	return &compat, nil
}

// RunComand to the fuse filesystem with chroot
func (c *CompatFS) RunCommand(command string) error {

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
