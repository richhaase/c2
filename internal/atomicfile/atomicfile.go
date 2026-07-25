package atomicfile

import (
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
)

func Write(path string, data []byte, perm fs.FileMode) error {
	dir := filepath.Dir(path)
	file, err := os.CreateTemp(dir, "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return err
	}
	tempPath := file.Name()
	keep := false
	defer func() {
		if !keep {
			_ = file.Close()
			_ = os.Remove(tempPath)
		}
	}()

	if err := file.Chmod(perm); err != nil {
		return err
	}
	n, err := file.Write(data)
	if err != nil {
		return err
	}
	if n != len(data) {
		return io.ErrShortWrite
	}
	if err := file.Sync(); err != nil {
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	if err := os.Rename(tempPath, path); err != nil {
		return err
	}
	keep = true
	return syncDir(dir)
}

func syncDir(dir string) error {
	if runtime.GOOS == "windows" {
		return nil
	}
	file, err := os.Open(dir)
	if err != nil {
		return err
	}
	defer func() { _ = file.Close() }()
	return file.Sync()
}
