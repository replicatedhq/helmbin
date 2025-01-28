package helpers

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
)

type MultiError struct {
	Errors []error
}

func (e *MultiError) Add(err error) {
	e.Errors = append(e.Errors, err)
}

func (e *MultiError) ErrorOrNil() error {
	switch len(e.Errors) {
	case 0:
		return nil
	case 1:
		return e.Errors[0]
	default:
		return fmt.Errorf("errors: %q", e.Errors)
	}
}

// MoveFile moves a file from one location to another, overwriting the destination if it
// exists. File mode is preserved.
func MoveFile(src, dst string) error {
	srcinfo, err := os.Stat(src)
	if err != nil {
		return fmt.Errorf("stat %s: %s", src, err)
	}

	if srcinfo.IsDir() {
		err := os.MkdirAll(dst, srcinfo.Mode())
		if err != nil {
			return fmt.Errorf("mkdir: %s", err)
		}

		err = os.Chmod(dst, srcinfo.Mode())
		if err != nil {
			return fmt.Errorf("chmod dir: %s", err)
		}

		entries, err := os.ReadDir(src)
		if err != nil {
			return fmt.Errorf("read source dir: %s", err)
		}
		for _, entry := range entries {
			err = MoveFile(filepath.Join(src, entry.Name()), filepath.Join(dst, entry.Name()))
			if err != nil {
				return fmt.Errorf("move file %s to %s: %s", entry.Name(), dst, err)
			}
		}
		return nil
	}

	srcfp, err := os.Open(src)
	if err != nil {
		return fmt.Errorf("open source file: %s", err)
	}
	defer srcfp.Close()

	opts := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	dstfp, err := os.OpenFile(dst, opts, srcinfo.Mode())
	if err != nil {
		return fmt.Errorf("open destination file: %s", err)
	}
	defer dstfp.Close()

	if _, err := io.Copy(dstfp, srcfp); err != nil {
		return fmt.Errorf("copy file: %s", err)
	}

	if err := dstfp.Sync(); err != nil {
		return fmt.Errorf("sync file: %s", err)
	}

	err = os.Chmod(dst, srcinfo.Mode())
	if err != nil {
		return fmt.Errorf("chmod file: %s", err)
	}

	if err := os.Remove(src); err != nil {
		return fmt.Errorf("remove source file: %s", err)
	}

	return nil
}

// RemoveAll removes path if it's a file. If path is a directory, it only removes its contents.
// This is to handle the case where path is a bind mounted directory.
func RemoveAll(path string) error {
	info, err := os.Lstat(path)
	if err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("stat file: %w", err)
	}
	if os.IsNotExist(err) {
		return nil
	}
	if !info.IsDir() {
		return os.Remove(path)
	}
	d, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("open directory: %w", err)
	}
	defer d.Close()
	names, err := d.Readdirnames(-1)
	if err != nil {
		return fmt.Errorf("read directory: %w", err)
	}
	var me MultiError
	for _, name := range names {
		if err := os.RemoveAll(filepath.Join(path, name)); err != nil {
			me.Add(fmt.Errorf("remove %s: %w", name, err))
		}
	}
	return me.ErrorOrNil()
}
