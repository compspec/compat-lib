package spindle

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"

	defaults "github.com/compspec/compat-lib/pkg/fs"
	"github.com/compspec/compat-lib/pkg/logger"
	"github.com/google/shlex"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// Generic update type to be parsed later
type Update struct {
	Message string
}

// Mock a cache, this is in memory to reflect the filesystem
// Once a record is added here, it's assumed present in <mountRoot>/cache/<path>
var cache = map[string]string{}

// Hack to store global mount point
var mountRoot = ""

type SpindleFS struct {
	Server *fuse.Server
	// Mountpoint has /root and /cache under it
	MountPoint string

	// Output file, if defined, to save events
	Outfile string
}

// RootFS returns the path in the root under the mountpoint
func (sfs *SpindleFS) RootFS() string {
	return filepath.Join(sfs.MountPoint, defaults.DefaultRootFS)
}

// CacheFS returns the path in the cache under the mountpoint
func (sfs *SpindleFS) CacheFS() string {
	return filepath.Join(sfs.MountPoint, defaults.DefaultCacheFS)
}

// Cleanup removes the mountpoint directory
func (sfs *SpindleFS) Cleanup(keepCache bool) {

	// Clean up mount point directory, including root and cache
	if !keepCache {
		fmt.Printf("Cleaning up %s...\n", sfs.MountPoint)
		os.RemoveAll(sfs.MountPoint)
	} else {
		// Just remove the /tmp/spindleXXX/root directory
		fmt.Printf("Keeping cache at %s...\n", sfs.CacheFS())
		os.RemoveAll(sfs.RootFS())
	}

	// Change permissions on output file
	if logger.Outfile != "" {
		fmt.Printf("Output file written to %s\n", logger.Outfile)
		os.Chmod(logger.Outfile, 0644)
	}
}

// MountedPath returns the path in the context of the fuse mount.
func (sfs *SpindleFS) MountedPath(path string) string {
	return filepath.Join(sfs.RootFS(), path)
}

// NewSpindleFS returns a new wrapper to a fuse.Server
// We mount a fusefs to a temporary directory
// The server returned (if not nil) needs to be
// correctly handled - see how it is used here in the library
// If recorder is true, we instantiate a recording base
func NewSpindleFS(
	mountPath string,
	recordFile string,
	readOnly bool,
) (*SpindleFS, error) {

	// Create a Compat Filesystem with defaults
	sfs := SpindleFS{Outfile: logger.Outfile}

	// Set the global log file in case we are recording events
	logger.SetOutfile(recordFile)

	// TODO keep track of cpu and memory profiles
	if mountPath == "" {
		mountPoint, err := os.MkdirTemp("", "spindle")
		if err != nil {
			return nil, err
		}
		mountPath = mountPoint
	}
	sfs.MountPoint = mountPath
	mountRoot = mountPath

	// Directories for the root, cache, and mount must exist
	for _, path := range []string{sfs.RootFS(), sfs.CacheFS(), mountPath} {
		_, err := os.Stat(path)
		if err != nil && os.IsNotExist(err) {
			err := os.Mkdir(path, 0755)
			if err != nil {
				return nil, err
			}
		}
	}

	// Create generic updates channel (this will eventually
	// be used for the service) and init faux cache
	updates := make(chan Update)
	cache = make(map[string]string)
	fmt.Printf("Mount directory %s\n", mountPath)

	// Mount the content of the rootFS (originalFS) at the mount point
	// Pass in a channel to receive updates from
	err := sfs.InitLoopbackRoot(
		defaults.OriginalFS,
		updates,
		readOnly,
	)
	if err != nil {
		return nil, err
	}
	return &sfs, nil
}

// RunComand to the fuse filesystem
func (sfs *SpindleFS) RunCommand(command, workdir string) error {

	// returns list of strings
	call, err := shlex.Split(command)
	if err != nil {
		return err
	}

	command, args := call[0], call[1:]

	// Place the working directory in context of the mount
	cwd := sfs.MountedPath(workdir)

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
