package slim

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"syscall"
	"time"

	defaults "github.com/compspec/compat-lib/pkg/fs"

	"github.com/compspec/compat-lib/pkg/utils"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// We need to implement a custom LoopbackNode Open function
var _ = (fs.NodeOpener)((*LoopbackNode)(nil))
var _ = (fs.NodeLookuper)((*LoopbackNode)(nil))
var _ = (fs.NodeFlusher)((*LoopbackNode)(nil))
var _ = (fs.NodeReadlinker)((*LoopbackNode)(nil))

type LoopbackNode struct {
	fs.LoopbackNode
}

func (n *LoopbackNode) path() string {
	path := n.Path(n.root())
	return filepath.Join(n.RootData.Path, path)
}

// fauxPath returns the read only path in the root
func (n *LoopbackNode) cachePath() string {
	path := n.Path(n.root())
	return filepath.Join(mountRoot, defaults.DefaultCacheFS, path)
}

// Lookup is the event when a path is being looked for. When it is found, then we see open.
func (n *LoopbackNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	originalPath := filepath.Join(n.path(), name)
	st := syscall.Stat_t{}
	err := syscall.Lstat(originalPath, &st)
	if err != nil {
		return nil, fs.ToErrno(err)
	}
	err = syscall.Stat(originalPath, &st)
	if err != nil {
		return nil, fs.ToErrno(err)
	}

	// If the lookup file exists, cache it
	// Commented out for now - does not work
	/*exists, _ := utils.PathExists(originalPath)
	if exists {
		_, ok := cache[originalPath]
		if !ok {
			cachePath := n.cachePath()
			utils.CopyFile(originalPath, cachePath)
			cache[originalPath] = cachePath
		}
	}*/
	out.Attr.FromStat(&st)
	node := newNode(n.RootData, n.EmbeddedInode(), name, &st)
	ch := n.NewInode(ctx, node, idFromStat(n.RootData, &st))
	return ch, 0
}

func (n *LoopbackNode) Readlink(ctx context.Context) ([]byte, syscall.Errno) {
	p := n.path()

	for l := 256; ; l *= 2 {
		buf := make([]byte, l)
		sz, err := syscall.Readlink(p, buf)
		fmt.Println(sz)
		if err != nil {
			return nil, fs.ToErrno(err)
		}
		if sz < len(buf) {
			return buf[:sz], 0
		}
	}
}

// Flush is called for the close(2) call, could be multiple times. See:
// https://github.com/hanwen/go-fuse/blob/aff07cbd88fef6a2561a87a1e43255516ba7d4b6/fs/api.go#L369
func (n *LoopbackNode) Flush(ctx context.Context, fh fs.FileHandle) syscall.Errno {
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
func (n *LoopbackNode) root() *fs.Inode {
	var rootNode *fs.Inode
	if n.RootData.RootNode != nil {
		rootNode = n.RootData.RootNode.EmbeddedInode()
	} else {
		rootNode = n.Root()
	}

	return rootNode
}

func (n *LoopbackNode) Open(ctx context.Context, flags uint32) (fs.FileHandle, uint32, syscall.Errno) {
	flags = flags &^ syscall.O_APPEND

	// Copy the open to our cache (and maintain directory structure)
	originalPath := n.path()
	cachePath, ok := cache[originalPath]
	if !ok {
		cachePath = n.cachePath()
		utils.CopyFile(originalPath, cachePath)
		cache[originalPath] = cachePath
	}

	// This next section emulates:
	// 	fh, flags, errno := n.LoopbackNode.Open(ctx, flags)
	// But we unwrap to get the fd (file descriptor) to uniquely identify
	fd, err := syscall.Open(cachePath, int(flags), 0)
	if err != nil {
		return nil, 0, fs.ToErrno(err)
	}

	//fmt.Printf("OpenSuccess %s %d\n", p, fd)
	loopbackFile := fs.NewLoopbackFile(fd)
	fh := &defaults.WrapperFile{
		AllFileOps: loopbackFile.(defaults.AllFileOps),
		Fid:        fd,
	}
	// fh, flags, errno
	return fh, 0, 0
}

func (n *LoopbackNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	inode, fh, flags, errno := n.LoopbackNode.Create(ctx, name, flags, mode, out)
	return inode, fh, flags, errno
}

func newNode(rootData *fs.LoopbackRoot, parent *fs.Inode, name string, st *syscall.Stat_t) fs.InodeEmbedder {
	n := &LoopbackNode{
		LoopbackNode: fs.LoopbackNode{
			RootData: rootData,
		},
	}
	return n
}

// InitLoopbackRoot creates a fuse.Server
func (sfs *SlimFS) InitLoopbackRoot(
	rootPath string,
	updates chan Update,
	readOnly bool,
) error {

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
			AllowOther:        defaults.Other,
			Debug:             defaults.Debug,
			DirectMount:       defaults.DirectMount,
			DirectMountStrict: defaults.DirectMountStrict,

			// First column in "df -T": original dir
			FsName: defaults.OriginalFS,

			// Second column in "df -T" will be shown as "fuse." + Name
			Name:   defaults.LoopbackName,
			Logger: log.New(os.Stderr, "", 0),
		},
	}

	// "read only"
	if readOnly {
		options.Options = []string{"ro"}
	}

	var st syscall.Stat_t
	err := syscall.Stat(rootPath, &st)
	if err != nil {
		return err
	}

	rootData := &fs.LoopbackRoot{
		NewNode: newNode,
		Path:    rootPath,
		Dev:     uint64(st.Dev),
	}

	// This is going to block
	server, err := fs.Mount(sfs.RootFS(), newNode(rootData, nil, "", &st), options)
	sfs.Server = server
	return err
}
