package fs

import (
	"context"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	"github.com/compspec/compat-lib/pkg/logger"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// We need to implement a custom LoopbackNode Open function
var _ = (fs.NodeOpener)((*CompatLoopbackNode)(nil))
var _ = (fs.NodeLookuper)((*CompatLoopbackNode)(nil))
var _ = (fs.NodeFlusher)((*CompatLoopbackNode)(nil))

type CompatLoopbackNode struct {
	fs.LoopbackNode
}

func (n *CompatLoopbackNode) path() string {
	path := n.Path(n.root())
	return filepath.Join(n.RootData.Path, path)
}

func (n *CompatLoopbackNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	p := filepath.Join(n.path(), name)
	st := syscall.Stat_t{}
	err := syscall.Lstat(p, &st)
	if err != nil {
		return nil, fs.ToErrno(err)
	}
	logger.LogEvent("Lookup", p)
	out.Attr.FromStat(&st)
	node := newNode(n.RootData, n.EmbeddedInode(), name, &st)
	ch := n.NewInode(ctx, node, idFromStat(n.RootData, &st))
	return ch, 0
}

// Flush is called for the close(2) call, could be multiple times. See:
// https://github.com/hanwen/go-fuse/blob/aff07cbd88fef6a2561a87a1e43255516ba7d4b6/fs/api.go#L369
func (n *CompatLoopbackNode) Flush(ctx context.Context, fh fs.FileHandle) syscall.Errno {
	p := n.path()
	logger.LogEvent("Close", p)
	return 0
}

// https://github.com/hanwen/go-fuse/blob/f5b6d1b67f4a4d0f4c3c88b4491185b3685e8383/fs/loopback.go#L48
func idFromStat(rootNode *fs.LoopbackRoot, st *syscall.Stat_t) fs.StableAttr {
	swapped := (uint64(st.Dev) << 32) | (uint64(st.Dev) >> 32)
	swappedRootDev := (rootNode.Dev << 32) | (rootNode.Dev >> 32)
	return fs.StableAttr{
		Mode: uint32(st.Mode),
		Gen:  1,
		// This should work well for traditional backing FSes,
		// not so much for other go-fuse FS-es
		Ino: (swapped ^ swappedRootDev) ^ st.Ino,
	}
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
	logger.LogEvent("Open", p)
	fh, flags, errno := n.LoopbackNode.Open(ctx, flags)
	return fh, flags, errno
}

func (n *CompatLoopbackNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	logger.LogEvent("Create", name)
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

// InitLoopbackRoot creates a fuse.Server
func (compat *CompatFS) InitLoopbackRoot(rootPath, mountPoint string, updates chan Update) error {

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
	compat.Server = server
	return err
}
