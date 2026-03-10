package refresh

import (
	"context"
	"os"

	"github.com/etnperlong/neko-webrtc-companion/internal/files"
)

// FileStore reads and writes the configuration content from disk.
type FileStore struct {
	path string
	perm os.FileMode
}

// NewFileStore returns a Store backed by the provided path and permissions.
// If perm is zero it defaults to 0600.
func NewFileStore(path string, perm os.FileMode) Store {
	if perm == 0 {
		perm = 0o600
	}
	return &FileStore{path: path, perm: perm}
}

// Read returns the stored data or nil when the file does not exist.
func (s *FileStore) Read(ctx context.Context) ([]byte, error) {
	data, err := os.ReadFile(s.path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	return data, nil
}

// Write persistently stores the content using an atomic writer.
func (s *FileStore) Write(ctx context.Context, data []byte) error {
	return files.WriteFileAtomic(s.path, data, s.perm)
}
