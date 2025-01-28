package helpers

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMoveFile(t *testing.T) {
	srcContent := []byte("test")
	srcFile, err := os.CreateTemp("", "source-*")
	assert.NoError(t, err)
	defer os.Remove(srcFile.Name())
	defer srcFile.Close()

	_, err = srcFile.Write(srcContent)
	assert.NoError(t, err)

	dstFile, err := os.CreateTemp("", "destination-*")
	assert.NoError(t, err)
	defer os.Remove(dstFile.Name())
	defer dstFile.Close()

	err = MoveFile(srcFile.Name(), dstFile.Name())
	assert.NoError(t, err)

	_, err = os.Stat(dstFile.Name())
	assert.NoError(t, err)

	content, err := os.ReadFile(dstFile.Name())
	assert.NoError(t, err)
	assert.Equal(t, srcContent, content)
}

func TestMoveFile_PreserveMode(t *testing.T) {
	srcContent := []byte("test")
	srcFile, err := os.CreateTemp("", "source-*")
	assert.NoError(t, err)
	defer os.Remove(srcFile.Name())
	defer srcFile.Close()

	_, err = srcFile.Write(srcContent)
	assert.NoError(t, err)

	err = os.Chmod(srcFile.Name(), 0755)
	assert.NoError(t, err)

	defer os.Remove("/tmp/testbin")
	err = MoveFile(srcFile.Name(), "/tmp/testbin")
	assert.NoError(t, err)

	info, err := os.Stat("/tmp/testbin")
	assert.NoError(t, err)
	assert.Equal(t, os.FileMode(0755), info.Mode())
}

func TestMoveFile_Directory(t *testing.T) {
	srcDir, err := os.MkdirTemp("", "sourcedir-*")
	assert.NoError(t, err)
	defer os.RemoveAll(srcDir)

	tmpDst, err := os.MkdirTemp("", "destination-*")
	assert.NoError(t, err)
	defer os.RemoveAll(tmpDst)

	dstDir := filepath.Join(tmpDst, "destination")

	err = os.Chmod(srcDir, 0777)
	assert.NoError(t, err)

	srcContent := []byte("test")
	srcFile, err := os.Create(filepath.Join(srcDir, "test.txt"))
	assert.NoError(t, err)
	defer os.Remove(srcFile.Name())
	defer srcFile.Close()

	_, err = srcFile.Write(srcContent)
	assert.NoError(t, err)

	err = MoveFile(srcDir, dstDir)
	assert.NoError(t, err)

	info, err := os.Stat(dstDir)
	assert.NoError(t, err)
	assert.Equal(t, info.IsDir(), true, "expected directory")
	assert.Equal(t, os.FileMode(0777|os.ModeDir).String(), info.Mode().String(), "unexpected file mode")

	_, err = os.Stat(filepath.Join(dstDir, "test.txt"))
	assert.NoError(t, err)

	content, err := os.ReadFile(filepath.Join(dstDir, "test.txt"))
	assert.NoError(t, err)
	assert.Equal(t, srcContent, content, "unexpected content")
}

func TestMoveFile_Symlink(t *testing.T) {
	srcFile, err := os.CreateTemp("", "source-*")
	assert.NoError(t, err)
	_, err = srcFile.Write([]byte("test"))
	assert.NoError(t, err)
	defer os.Remove(srcFile.Name())
	defer srcFile.Close()

	symlinkPath := fmt.Sprintf("%s-symlink", srcFile.Name())
	err = os.Symlink(srcFile.Name(), symlinkPath)
	assert.NoError(t, err)
	defer os.Remove(symlinkPath)

	err = MoveFile(symlinkPath, "/tmp/destination")
	assert.NoError(t, err)
	defer os.RemoveAll("/tmp/destination")

	target, err := os.Readlink(symlinkPath)
	assert.Error(t, err)
	assert.Empty(t, target)
}

func TestRemoveAll(t *testing.T) {
	tests := []struct {
		name  string
		setup func(t *testing.T) (string, bool)
		isDir bool
	}{
		{
			name: "remove file",
			setup: func(t *testing.T) (string, bool) {
				f, err := os.CreateTemp("", "test-file")
				if err != nil {
					t.Fatal(err)
				}
				return f.Name(), true
			},
		},
		{
			name: "remove directory",
			setup: func(t *testing.T) (string, bool) {
				dir, err := os.MkdirTemp("", "test-dir")
				if err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "file1"), []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
				if err := os.Mkdir(filepath.Join(dir, "subdir"), 0755); err != nil {
					t.Fatal(err)
				}
				if err := os.WriteFile(filepath.Join(dir, "subdir", "file2"), []byte("test"), 0644); err != nil {
					t.Fatal(err)
				}
				return dir, true
			},
			isDir: true,
		},
		{
			name: "remove symlink",
			setup: func(t *testing.T) (string, bool) {
				f, err := os.CreateTemp("", "test-file")
				if err != nil {
					t.Fatal(err)
				}
				slink := filepath.Join(os.TempDir(), "test-symlink")
				if err := os.Symlink(f.Name(), slink); err != nil {
					t.Fatal(err)
				}
				return slink, true
			},
		},
		{
			name: "remove non-existent path",
			setup: func(t *testing.T) (string, bool) {
				return filepath.Join(os.TempDir(), "non-existent-path"), false
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := require.New(t)

			path, shouldExist := tt.setup(t)
			_, err := os.Lstat(path)
			if shouldExist {
				req.NoError(err)
			} else {
				req.Error(err)
			}

			if tt.isDir {
				// validate dir has contents
				d, err := os.Open(path)
				req.NoError(err)
				defer d.Close()

				names, err := d.Readdirnames(-1)
				req.NoError(err)
				req.NotEmpty(names)
			}

			// remove the path
			err = RemoveAll(path)
			req.NoError(err)

			if !tt.isDir {
				// file should be gone
				_, err := os.Lstat(path)
				req.Error(err)
			} else {
				// dir should exist and be empty
				_, err := os.Lstat(path)
				req.NoError(err)

				d, err := os.Open(path)
				req.NoError(err)
				defer d.Close()

				names, err := d.Readdirnames(-1)
				req.NoError(err)
				req.Empty(names)
			}
		})
	}
}
