package fs

import "os"

// ArchiveType defines the types of supported archives..
type archiveType string
type compressionType string

const (
	ZIP archiveType = "zip"
	TAR archiveType = "tar"

	BR     compressionType = "br"
	BZ2    compressionType = "bz2"
	GZIP   compressionType = "gz"
	LZ4    compressionType = "lz4"
	LZ     compressionType = "lz"
	SNAPPY compressionType = "sz"
	S2     compressionType = "s2"
	XZ     compressionType = "xz"
	ZZ     compressionType = "zz"
	ZST    compressionType = "zst"
)

// Archive interface defines operations for archive management..
type Archive interface {
	Path() string
	AddFile(filePath string) error
	AddDirectory(dirPath string) error
	Compress() error
}

type ArchiveOptions struct {
	CompressionType compressionType
	ArchiveType     archiveType
	Level           int
}

type archive struct {
	options ArchiveOptions
	path    string
	entries []System
}

// NewArchive creates a new Archive object based on the given options..
func NewArchive(path string, options ArchiveOptions) Archive {
	return &archive{
		options: options,
		path:    path,
	}
}

// Path returns the path of the archive..
func (a *archive) Path() string {
	return a.path
}

// AddFile adds a file to the archive..
func (a *archive) AddFile(filePath string) error {
	f, err := NewFile(Option{
		Path: filePath,
	})

	if err != nil {
		return err
	}

	if !f.Exists() {
		return os.ErrNotExist
	}

	for _, e := range a.entries {
		if e.Path() == filePath {
			return nil
		}

		if d, ok := e.(Directory); ok {
			if d.Has(filePath) {
				return nil
			}
		}
	}

	a.entries = append(a.entries, f)

	return nil
}

// AddDirectory adds a directory to the archive..
func (a *archive) AddDirectory(dirPath string) error {
	d, err := NewDirectory(Option{
		Path: dirPath,
	})

	if err != nil {
		return err
	}

	if !d.Exists() {
		return os.ErrNotExist
	}

	for _, e := range a.entries {
		if e.Path() == dirPath {
			return nil
		}

		if d, ok := e.(Directory); ok {
			if d.Has(dirPath) {
				return nil
			}
		}
	}

	a.entries = append(a.entries, d)

	return nil
}

// Compress compresses the archive..
func (a *archive) Compress() error {
	return nil
}
