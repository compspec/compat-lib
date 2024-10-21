package fs

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// We need to implement a custom LoopbackNode Open function
var _ = (fs.NodeOpener)((*CompatLoopbackNode)(nil))

type CompatLoopbackNode struct {
	fs.LoopbackNode
}

func (n *CompatLoopbackNode) path() string {
	path := n.Path(n.root())
	return filepath.Join(n.RootData.Path, path)
}

// path returns the full path to the file in the underlying file system.
func (n *CompatLoopbackNode) root() *fs.Inode {
	var rootNode *fs.Inode
	if n.RootData.RootNode != nil {
		rootNode = n.RootData.RootNode.EmbeddedInode()
	} else {
		rootNode = n.Root()
	}

	return rootNode
}

func (n *CompatLoopbackNode) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	flags = flags &^ syscall.O_APPEND
	p := n.path()
	fmt.Printf("CUSTOM OPEN FOR %s with flags %d\n", p, flags)
	fh, flags, errno := n.LoopbackNode.Open(ctx, flags)
	return fh, flags, errno
}

func (n *CompatLoopbackNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	fmt.Printf("CUSTOM CREATE FOR %s with flags %d\n", name, flags)
	inode, fh, flags, errno := n.LoopbackNode.Create(ctx, name, flags, mode, out)
	return inode, fh, flags, errno
}

func newNode(rootData *fs.LoopbackRoot, parent *fs.Inode, name string, st *syscall.Stat_t) fs.InodeEmbedder {
	n := &CompatLoopbackNode{
		LoopbackNode: fs.LoopbackNode{
			RootData: rootData,
		},
	}
	return n
}

func NewLoopbackRoot(rootPath, mountPoint string) (*fuse.Server, error) {

	rootData := &fs.LoopbackRoot{
		NewNode: newNode,
		Path:    rootPath,
	}

	// one second is compatible with libfuse defaults
	// https://man7.org/linux/man-pages/man8/mount.fuse3.8.html
	oneSecond := time.Second

	// https://github.com/hanwen/go-fuse/blob/master/fs/api.go
	options := &fs.Options{
		AttrTimeout:  &oneSecond,
		EntryTimeout: &oneSecond,
		// Leave file permissions on "000" files
		NullPermissions: true,
		MountOptions: fuse.MountOptions{
			AllowOther:        other,
			Debug:             debug,
			DirectMount:       directMount,
			DirectMountStrict: directMountStrict,

			// First column in "df -T": original dir
			FsName: originalFS,

			// Second column in "df -T" will be shown as "fuse." + Name
			Name: "loopback",

			// "read only"
			Options: []string{"ro"},
			Logger:  log.New(os.Stderr, "", 0),
		},
	}

	// This is  going to block
	server, err := fs.Mount(mountPoint, newNode(rootData, nil, "", nil), options)
	if err != nil {
		return nil, err
	}
	return server, nil
}
