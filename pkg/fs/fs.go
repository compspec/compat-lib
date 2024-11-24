package fs

// Defaults across filesystem types

const (
	Debug = false

	// mount with -o allowother
	Other = false

	// Try to use "mount" syscall instead of fusermount
	DirectMount = false

	// Allow to fall back to fusermount (probably doesn't matter given directMount false)
	DirectMountStrict = false

	// original FS is for the loopback root
	OriginalFS = "/"

	// Writes to /tmp/compatlibxxxx
	TemporaryDirname = "compatlib"

	// Name for the loopback filesystem
	LoopbackName = "loopback"

	// Default FS root under the mountpoint
	DefaultRootFS = "root"

	// Default FS cache under the mountpoint
	DefaultCacheFS = "cache"
)
