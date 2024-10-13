package fs

import (
	"fmt"
	"os"

	"github.com/hanwen/go-fuse/v2/fuse"
)

// Keep these constants for now
const (
	debug = true

	// mount with -o allowother
	other = false

	// Try to use "mount" syscall instead of fusermount
	directMount = false

	// Allow to fall back to fusermount (probably doesn't matter given directMount false)
	directMountStrict = false

	// original FS is for the loopback root
	originalFS = "/"
)

type CompatFS struct {
	Server     *fuse.Server
	MountPoint string
}

// Cleanup removes the mountpoint directory
func (c *CompatFS) Cleanup() {

	// Clean up mount point directory
	fmt.Printf("Cleaning up %s...\n", c.MountPoint)
	os.RemoveAll(c.MountPoint)
}

// NewCompatFS returns a new wrapper to a fuse.Server
// We mount a fusefs to a temporary directory
// The server returned (if not nil) needs to be
// correctly handled - see how it is used here in the library
func NewCompatFS() (*CompatFS, error) {

	// Create a Compat Filesystem with defaults
	compat := CompatFS{}

	// TODO keep track of cpu and memory profiles

	mountPoint, err := os.MkdirTemp("", "compat-lib")
	if err != nil {
		return nil, err
	}
	fmt.Printf("Mount directory %s\n", mountPoint)
	compat.MountPoint = mountPoint

	// TODO add cpu / memory monitor
	//    fname := filepath.Join(dname, "file1")
	//  err = os.WriteFile(fname, []byte{1, 2}, 0666)
	//  check(err)
	server, err := NewLoopbackRoot(originalFS, mountPoint)
	if err != nil {
		return nil, err
	}
	compat.Server = server
	return &compat, nil
}
