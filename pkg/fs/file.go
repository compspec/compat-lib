package fs

import (
	"github.com/hanwen/go-fuse/v2/fs"
)

// The custom Wrapper file allows us to carry forward the file handle
type AllFileOps interface {
	fs.FileReader
	fs.FileWriter
	fs.FileFlusher
}

// We use this wrapperFile type to hold the file handle
// This way we can make a direct association between open and close
type WrapperFile struct {
	AllFileOps
	Fid int
}
