package fs_test

import (
	"fmt"
	"os"
	"os/user"
	"strconv"
	"testing"

	"github.com/kitsunium/sdk/pkg/kernel/fs"
	"github.com/kitsunium/sdk/pkg/lib/pointer"
	"github.com/stretchr/testify/assert"
)

func TestFile(t *testing.T) {
	tmpFile, err := os.CreateTemp("", "testfile")
	assert.NoError(t, err)
	tmpDir, err := os.MkdirTemp("", "testdir")
	assert.NoError(t, err)

	defer os.Remove(tmpFile.Name())
	defer os.RemoveAll(tmpDir)

	t.Run("NewFile", func(t *testing.T) {
		t.Run("nominal", func(t *testing.T) {
			f, err := fs.NewFile(fs.Option{
				Path: tmpFile.Name(),
			})
			assert.NoError(t, err)
			assert.NotNil(t, f)
		})

		t.Run("path empty", func(t *testing.T) {
			f, err := fs.NewFile(fs.Option{})
			assert.Error(t, err)
			assert.Nil(t, f)
		})

		t.Run("path is only space", func(t *testing.T) {
			f, err := fs.NewFile(fs.Option{
				Path: " ",
			})
			assert.Error(t, err)
			assert.Nil(t, f)
		})

		t.Run("path with space", func(t *testing.T) {
			f, err := fs.NewFile(fs.Option{
				Path: " " + tmpFile.Name() + " ",
			})
			assert.NoError(t, err)
			assert.NotNil(t, f)
		})

		t.Run("create if not exist", func(t *testing.T) {
			os.Remove(tmpFile.Name())
			f, err := fs.NewFile(fs.Option{
				Path:             tmpFile.Name(),
				CreateIfNotExist: true,
			})
			assert.NoError(t, err)
			assert.NotNil(t, f)
		})

		t.Run("create if not exist", func(t *testing.T) {
			f, err := fs.NewFile(fs.Option{
				Path:             tmpFile.Name(),
				CreateIfNotExist: true,
			})
			assert.NoError(t, err)
			assert.NotNil(t, f)
		})

		t.Run("is not a file", func(t *testing.T) {
			f, err := fs.NewFile(fs.Option{
				Path: tmpDir,
			})
			assert.Error(t, err)
			assert.Nil(t, f)
		})

		t.Run("is Symbolic link", func(t *testing.T) {
			origin := tmpDir + "/original"
			symlink := tmpDir + "/symlink"

			f, err := fs.NewFile(fs.Option{
				Path:             origin,
				CreateIfNotExist: true,
			})

			os.Symlink(origin, symlink)

			f, err = fs.NewFile(fs.Option{
				Path: symlink,
			})

			assert.NoError(t, err)
			assert.NotNil(t, f)
		})

		t.Run("is dot file", func(t *testing.T) {
			dotFile := tmpDir + "/.dotfile"

			f, err := fs.NewFile(fs.Option{
				Path: dotFile,
			})

			assert.NoError(t, err)
			assert.NotNil(t, f)

			assert.True(t, f.IsDotFile())
		})
	})

	t.Run("Path", func(t *testing.T) {
		f, err := fs.NewFile(fs.Option{
			Path: tmpFile.Name(),
		})
		assert.NoError(t, err)
		assert.NotNil(t, f)

		t.Run("nominal", func(t *testing.T) {
			assert.Equal(t, tmpFile.Name(), f.Path())
		})
	})

	t.Run("Size", func(t *testing.T) {
		f, err := fs.NewFile(fs.Option{
			Path: tmpFile.Name(),
		})

		assert.NoError(t, err)
		assert.NotNil(t, f)

		t.Run("nominal", func(t *testing.T) {
			assert.Equal(t, int64(0), f.Size())
		})

		f.Write([]byte("test"))

		t.Run("after write", func(t *testing.T) {
			assert.Equal(t, int64(4), f.Size())
		})
	})

	t.Run("Parent", func(t *testing.T) {
		f, err := fs.NewFile(fs.Option{
			Path:             tmpFile.Name(),
			CreateIfNotExist: true,
		})
		assert.NoError(t, err)
		assert.NotNil(t, f)

		parent, err := f.Parent()

		assert.NoError(t, err)
		assert.NotNil(t, parent)
	})

	t.Run("Remove", func(t *testing.T) {
		f, err := fs.NewFile(fs.Option{
			Path:             tmpFile.Name(),
			CreateIfNotExist: true,
		})
		assert.NoError(t, err)
		assert.NotNil(t, f)

		t.Run("nominal", func(t *testing.T) {
			err = f.Remove()
			assert.NoError(t, err)

			_, err = os.Stat(tmpFile.Name())
			assert.Error(t, err)
		})

		t.Run("file not exist", func(t *testing.T) {
			err = f.Remove()
			assert.Error(t, err)
		})
	})

	t.Run("Exists", func(t *testing.T) {
		f, err := fs.NewFile(fs.Option{
			Path:             tmpFile.Name(),
			CreateIfNotExist: true,
		})

		assert.NoError(t, err)
		assert.NotNil(t, f)

		t.Run("nominal", func(t *testing.T) {
			assert.True(t, f.Exists())
		})

		t.Run("file not exist", func(t *testing.T) {
			err := os.Remove(tmpFile.Name())
			assert.NoError(t, err)
			assert.False(t, f.Exists())
		})
	})

	t.Run("Create", func(t *testing.T) {
		f, err := fs.NewFile(fs.Option{
			Path: tmpFile.Name(),
		})
		assert.NoError(t, err)
		assert.NotNil(t, f)

		defer os.Remove(tmpFile.Name())

		t.Run("nominal", func(t *testing.T) {
			f, err = f.Create()
			assert.NoError(t, err)
			assert.NotNil(t, f)
		})

		t.Run("file already exist", func(t *testing.T) {
			f, err = f.Create()
			assert.NoError(t, err)
			assert.NotNil(t, f)
		})

		t.Run("file with custom permission", func(t *testing.T) {
			os.Remove(tmpFile.Name())
			f, err = fs.NewFile(fs.Option{
				Path:  tmpFile.Name(),
				Chmod: pointer.Uint32(0777),
			})
		})

		t.Run("file with custom authorized uid/gid", func(t *testing.T) {
			user, err := user.Current()
			assert.NoError(t, err)
			gid, err := strconv.Atoi(user.Gid)
			assert.NoError(t, err)
			uid, err := strconv.Atoi(user.Uid)
			assert.NoError(t, err)

			os.Remove(tmpFile.Name())
			f, err = fs.NewFile(fs.Option{
				Path:             tmpFile.Name(),
				UID:              pointer.Int(uid),
				GID:              pointer.Int(gid),
				CreateIfNotExist: true,
			})

			assert.NoError(t, err)
			assert.NotNil(t, f)
		})

		t.Run("file with custom unauthorized uid/gid", func(t *testing.T) {
			user, err := user.Lookup("root")
			assert.NoError(t, err)
			gid, err := strconv.Atoi(user.Gid)
			assert.NoError(t, err)
			uid, err := strconv.Atoi(user.Uid)
			assert.NoError(t, err)

			os.Remove(tmpFile.Name())
			f, err = fs.NewFile(fs.Option{
				Path:             tmpFile.Name(),
				UID:              pointer.Int(uid),
				GID:              pointer.Int(gid),
				CreateIfNotExist: true,
			})

			assert.Error(t, err)
			assert.Nil(t, f)
		})

		t.Run("file with invalid path", func(t *testing.T) {
			invalidPath := "/invalid-path/testfile"
			f, err := fs.NewFile(fs.Option{
				Path:             invalidPath,
				CreateIfNotExist: true,
			})

			assert.Error(t, err, "expected error for invalid path")
			assert.Nil(t, f, "file object should be nil for invalid path")
		})

		t.Run("file with directory path", func(t *testing.T) {
			f, err := fs.NewFile(fs.Option{
				Path:             tmpDir, // tmpDir est un répertoire
				CreateIfNotExist: true,
			})

			assert.Error(t, err, "expected error for directory path")
			assert.Nil(t, f, "file object should be nil for directory path")
		})

		t.Run("overwrite", func(t *testing.T) {
			f, err := fs.NewFile(fs.Option{
				Path:             tmpFile.Name(),
				CreateIfNotExist: true,
			})

			assert.NoError(t, err, "unexpected error creating file")
			assert.NotNil(t, f, "file object should not be nil")

			// Écrire du contenu dans le fichier
			_, err = f.Write([]byte("test"))
			assert.NoError(t, err, "unexpected error writing to file")

			// Vérifier que le contenu est écrit
			bts, err := f.Read()
			assert.NoError(t, err, "unexpected error reading file")
			assert.Equal(t, "test", string(bts), "file content mismatch after write")

			// Recréer avec Overwrite activé
			f, err = fs.NewFile(fs.Option{
				Path:      tmpFile.Name(),
				Overwrite: true,
			})

			assert.NoError(t, err, "unexpected error recreating file with overwrite")
			assert.NotNil(t, f, "file object should not be nil after recreate")

			f, err = f.Create()
			assert.NoError(t, err, "unexpected error recreating file with overwrite")
			assert.NotNil(t, f, "file object should not be nil after recreate")

			// Lire à nouveau et vérifier que le fichier est vide
			bts, err = f.Read()
			assert.NoError(t, err, "unexpected error reading file after overwrite")
			assert.Empty(t, bts, "file content should be empty after overwrite")
			os.Remove(tmpFile.Name())
		})
	})

	t.Run("Copy", func(t *testing.T) {
		t.Run("nominal", func(t *testing.T) {
			content := []byte("Hello, World!")
			srcFile, err := os.CreateTemp("", "srcfile")
			assert.NoError(t, err)
			os.Remove(srcFile.Name())

			f, err := fs.NewFile(fs.Option{
				Path:             srcFile.Name(),
				CreateIfNotExist: true,
			})
			assert.NoError(t, err)

			f.Write(content)

			dstFile := srcFile.Name() + "_copy"
			err = f.Copy(dstFile)
			assert.NoError(t, err)
			defer os.Remove(dstFile)

			data, err := f.Read()
			assert.NoError(t, err)
			assert.Equal(t, content, data, "file content mismatch after copy")
		})

		t.Run("nonexistent source", func(t *testing.T) {
			nonExistentPath := "/nonexistent/path/to/file"
			dstFile := tmpDir + "/nonexistent_copy"

			f, err := fs.NewFile(fs.Option{
				Path: nonExistentPath,
			})
			assert.NoError(t, err)

			err = f.Copy(dstFile)
			assert.Error(t, err, "copying from a nonexistent file should return an error")
		})

		t.Run("invalid destination", func(t *testing.T) {
			srcFile, err := os.CreateTemp("", "srcfile")
			assert.NoError(t, err)
			defer os.Remove(srcFile.Name())

			content := []byte("Hello, World!")
			_, err = srcFile.Write(content)
			assert.NoError(t, err)
			srcFile.Close()

			invalidPath := "/invalid/path/to/copy"

			f, err := fs.NewFile(fs.Option{
				Path: srcFile.Name(),
			})
			assert.NoError(t, err)

			err = f.Copy(invalidPath)
			assert.Error(t, err, "copying to an invalid path should return an error")
		})
	})

	t.Run("Move", func(t *testing.T) {
		t.Run("nominal", func(t *testing.T) {
			srcFile, err := os.CreateTemp("", "srcfile")
			assert.NoError(t, err)
			defer os.Remove(srcFile.Name())

			content := []byte("Hello, World!")
			_, err = srcFile.Write(content)
			assert.NoError(t, err)
			srcFile.Close()

			dstFile := srcFile.Name() + "_moved"

			f, err := fs.NewFile(fs.Option{
				Path: srcFile.Name(),
			})
			assert.NoError(t, err)

			err = f.Move(dstFile)
			assert.NoError(t, err)
			defer os.Remove(dstFile)

			_, err = os.Stat(srcFile.Name())
			assert.Error(t, err, "source file should not exist after move")

			data, err := os.ReadFile(dstFile)
			assert.NoError(t, err)
			assert.Equal(t, content, data, "file content mismatch after move")
		})

		t.Run("nonexistent source", func(t *testing.T) {
			nonExistentPath := "/nonexistent/path/to/file"
			dstFile := tmpDir + "/nonexistent_moved"

			f, err := fs.NewFile(fs.Option{
				Path: nonExistentPath,
			})
			assert.NoError(t, err)

			err = f.Move(dstFile)
			assert.Error(t, err, "moving a nonexistent file should return an error")
		})

		t.Run("invalid destination", func(t *testing.T) {
			srcFile, err := os.CreateTemp("", "srcfile")
			assert.NoError(t, err)
			defer os.Remove(srcFile.Name())

			content := []byte("Hello, World!")
			_, err = srcFile.Write(content)
			assert.NoError(t, err)
			srcFile.Close()

			invalidPath := "/invalid/path/to/move"

			f, err := fs.NewFile(fs.Option{
				Path: srcFile.Name(),
			})
			assert.NoError(t, err)

			err = f.Move(invalidPath)
			assert.Error(t, err, "moving to an invalid path should return an error")
		})
	})

}

func BenchmarkFileCopy(b *testing.B) {
	tmpSrcFile, err := os.CreateTemp("", "srcfile")
	assert.NoError(b, err)
	defer os.Remove(tmpSrcFile.Name())

	tmpDstFile := tmpSrcFile.Name() + ".copy"
	defer os.Remove(tmpDstFile)

	// Write some data to the source file
	data := make([]byte, 1*1024*1024) // 1 MB data
	_, err = tmpSrcFile.Write(data)
	assert.NoError(b, err)
	tmpSrcFile.Close()

	bufferSizes := []int{4 * 1024, 8 * 1024, 16 * 1024, 32 * 1024, 64 * 1024, 128 * 1024, 256 * 1024, 512 * 1024, 1 * 1024 * 1024, 2 * 1024 * 1024, 4 * 1024 * 1024, 8 * 1024 * 1024, 16 * 1024 * 1024, 32 * 1024 * 1024}

	for _, bufferSize := range bufferSizes {
		b.Run("BufferSize="+fmt.Sprint(bufferSize/1024), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				f, err := fs.NewFile(fs.Option{
					Path:             tmpSrcFile.Name(),
					BufferSize:       &bufferSize,
					CreateIfNotExist: true,
				})
				assert.NoError(b, err)

				err = f.Copy(tmpDstFile + strconv.Itoa(i))
				assert.NoError(b, err)
			}

			os.RemoveAll(tmpDstFile)
		})
	}
}
