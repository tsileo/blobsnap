package store

import (
	"os"
	"path"
)

type FileBlobStore struct {
	dir string
}

func NewFileBlobStore(dir string) FileBlobStore {
	return FileBlobStore{dir}
}

func (fs FileBlobStore) fileName(hash string) string {
	return path.Join(fs.dir, hash)
}

func (fs FileBlobStore) Stat(hash string) (bool, error) {
	if _, err := os.Stat(fs.fileName(hash)); os.IsNotExist(err) {
		return false, nil
	} else if err != nil {
		return false, err
	}
	return true, nil
}

func (fs FileBlobStore) Get(hash string) ([]byte, error) {
	return os.ReadFile(fs.fileName(hash))
}

func (fs FileBlobStore) Put(hash string, data []byte) error {
	return os.WriteFile(fs.fileName(hash), data, 0644)
}

func (fs FileBlobStore) Close() {
	return
}
