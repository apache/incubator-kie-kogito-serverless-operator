package resources

import (
	"io"
	"os"
	"path/filepath"
)

// CloseQuietly unconditionally close an io.Closer
// It should not be used to replace the Close statement(s).
func CloseQuietly(closer io.Closer) {
	_ = closer.Close()
}

// Open a safe wrapper of os.Open.
func Open(name string) (*os.File, error) {
	return os.Open(filepath.Clean(name))
}
