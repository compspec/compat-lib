package record

import (
	"context"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"syscall"
	"time"
	"unsafe"

	defaults "github.com/compspec/compat-lib/pkg/fs"

	"github.com/compspec/compat-lib/pkg/logger"
	"github.com/hanwen/go-fuse/v2/fs"
	"github.com/hanwen/go-fuse/v2/fuse"
)

// We need to implement a custom LoopbackNode Open function
var _ = (fs.NodeOpener)((*LoopbackNode)(nil))
var _ = (fs.NodeLookuper)((*LoopbackNode)(nil))
var _ = (fs.NodeFlusher)((*LoopbackNode)(nil))

type LoopbackNode struct {
	fs.LoopbackNode
}

func (n *LoopbackNode) path() string {
	path := n.Path(n.root())
	return filepath.Join(n.RootData.Path, path)
}

type AllFileOps interface {
	fs.FileReader
	fs.FileWriter
	fs.FileFlusher
}

// We use this wrapperFile type to hold the file handle
// This way we can make a direct association between open and close
type wrapperFile struct {
	AllFileOps
	Fid int
}

func (n *LoopbackNode) Lookup(ctx context.Context, name string, out *fuse.EntryOut) (*fs.Inode, syscall.Errno) {
	p := filepath.Join(n.path(), name)
	st := syscall.Stat_t{}
	err := syscall.Lstat(p, &st)
	if err != nil {
		return nil, fs.ToErrno(err)
	}
	// Stat has the following:
	// Dev     Ino      Nlink Mode  Uid Gid X_pad Rdev Size  Blksize Blocks  Atim          Mtim           Ctim                   X_unused
	//{2097217 13408317 1     41471 0   0   0     0    18    4096    0      {1633012128 0} {1633012128 0} {1732061992 277100520} [0 0 0]}
	logger.LogEvent("Lookup", p)
	out.Attr.FromStat(&st)
	node := newNode(n.RootData, n.EmbeddedInode(), name, &st)
	ch := n.NewInode(ctx, node, idFromStat(n.RootData, &st))
	return ch, 0
}

func GetUnexportedField(field reflect.Value) interface{} {
	return reflect.NewAt(field.Type(), unsafe.Pointer(field.UnsafeAddr())).Elem().Interface()
}

// Flush is called for the close(2) call, could be multiple times. See:
// https://github.com/hanwen/go-fuse/blob/aff07cbd88fef6a2561a87a1e43255516ba7d4b6/fs/api.go#L369
func (n *LoopbackNode) Flush(ctx context.Context, fh fs.FileHandle) syscall.Errno {
	p := n.path()
	wf, ok := fh.(*wrapperFile)
	if !ok {
		fmt.Printf("Warning: cannot serialize %s back to wrapped file, this should not happen\n", p)
	}
	logger.LogEvent("Close", fmt.Sprintf("%s\t%d", p, wf.Fid))
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
	p := n.path()

	// This next section emulates:
	// 	fh, flags, errno := n.LoopbackNode.Open(ctx, flags)
	// But we unwrap to get the fd (file descriptor) to uniquely identify
	fd, err := syscall.Open(p, int(flags), 0)
	if err != nil {
		return nil, 0, fs.ToErrno(err)
	}
	logger.LogEvent("Open", fmt.Sprintf("%s\t%d", p, fd))
	loopbackFile := fs.NewLoopbackFile(fd)
	fh := &wrapperFile{
		AllFileOps: loopbackFile.(AllFileOps),
		Fid:        fd,
	}

	// fh, flags, errno
	return fh, 0, 0
}

func (n *LoopbackNode) Create(ctx context.Context, name string, flags uint32, mode uint32, out *fuse.EntryOut) (*fs.Inode, fs.FileHandle, uint32, syscall.Errno) {
	logger.LogEvent("Create", name)
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
func (rfs *RecordFS) InitLoopbackRoot(
	rootPath, mountPoint string,
	updates chan Update,
	readOnly bool,
) error {

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

	// This is  going to block
	server, err := fs.Mount(mountPoint, newNode(rootData, nil, "", nil), options)
	rfs.Server = server
	return err
}
