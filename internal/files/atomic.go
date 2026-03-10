package files

import (
	"os"
	"path/filepath"
)

// WriteFileAtomic writes data to path using a temporary file within the same
// directory before renaming it over path to ensure atomicity. The final file
// always has the requested permissions.
func WriteFileAtomic(path string, data []byte, perm os.FileMode) error {
	dir := filepath.Dir(path)
	tmp, err := os.CreateTemp(dir, ".tmpfile-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	var renamed bool
	defer func() {
		if tmp != nil {
			tmp.Close()
		}
		if !renamed {
			os.Remove(tmpPath)
		}
	}()

	if _, err := tmp.Write(data); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Chmod(perm); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	tmp = nil

	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	renamed = true
	return nil
}
