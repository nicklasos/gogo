package unit

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"app/internal/uploads"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLocalStorage_Resolve(t *testing.T) {
	root := t.TempDir()
	storage := uploads.NewLocalStorage(root, "http://files.test/api/files/")

	inside, err := storage.Resolve("12/photo.jpg")
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "12", "photo.jpg"), inside)

	for _, path := range []string{"../secret.txt", "12/../../secret.txt", "/../../etc/passwd", "..", "../" + filepath.Base(root) + "-sibling/file"} {
		resolved, err := storage.Resolve(path)
		if err == nil {
			assert.True(t, strings.HasPrefix(resolved, root+string(os.PathSeparator)), "%q resolved outside the root: %s", path, resolved)
		}
	}

	for _, path := range []string{"", "/", "."} {
		_, err := storage.Resolve(path)
		assert.ErrorIs(t, err, uploads.ErrInvalidPath, "%q is the root itself, not a file", path)
	}
}

func TestLocalStorage_SaveDeleteURL(t *testing.T) {
	root := t.TempDir()
	storage := uploads.NewLocalStorage(root, "http://files.test/api/files/")
	ctx := context.Background()

	require.NoError(t, storage.Save(ctx, "7/a.txt", strings.NewReader("hello")))
	content, err := os.ReadFile(filepath.Join(root, "7", "a.txt"))
	require.NoError(t, err)
	assert.Equal(t, "hello", string(content))

	assert.Equal(t, "http://files.test/api/files/7/a.txt", storage.URL("7/a.txt"))
	assert.Equal(t, "http://files.test/api/files/7/a.txt", storage.URL("/7/a.txt"))
	assert.Equal(t, "", storage.URL(""))

	outside := filepath.Join(filepath.Dir(root), "outside-"+filepath.Base(root)+".txt")
	require.NoError(t, storage.Save(ctx, "../outside-"+filepath.Base(root)+".txt", strings.NewReader("x")))
	_, err = os.Stat(outside)
	assert.True(t, os.IsNotExist(err), "a traversal path must be written inside the root, never beside it")

	require.NoError(t, storage.Delete(ctx, "7/a.txt"))
	require.NoError(t, storage.Delete(ctx, "7/a.txt"), "deleting a missing file is not an error")
	_, err = os.Stat(filepath.Join(root, "7", "a.txt"))
	assert.True(t, os.IsNotExist(err))
}
