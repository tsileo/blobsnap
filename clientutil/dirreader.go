package clientutil

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dchest/blake2b"

	"github.com/antonovvk/blobsnap/store"
)

// GetDir restore the directory to path
func GetDir(bs store.BlobStore, key, path string) (rr *ReadResult, err error) {
	fullHash := blake2b.New256()
	rr = &ReadResult{}
	err = os.Mkdir(path, 0700)
	if err != nil {
		return
	}
	meta, err := NewMetaFromBlobStore(bs, key)
	if err != nil {
		return nil, fmt.Errorf("failed to fetch meta: %v", err)
	}
	meta.Hash = key
	var crr *ReadResult
	if meta.Size > 0 {
		for _, hash := range meta.Refs {
			meta, err := NewMetaFromBlobStore(bs, hash.(string))
			if err != nil {
				return nil, fmt.Errorf("failed to fetch meta: %v", err)
			}
			if meta.IsFile() {
				crr, err = GetFile(bs, meta.Hash, filepath.Join(path, meta.Name))
				if err != nil {
					return rr, fmt.Errorf("failed to GetFile %+v: %v", meta, err)
				}
			} else {
				crr, err = GetDir(bs, meta.Hash, filepath.Join(path, meta.Name))
				if err != nil {
					return rr, fmt.Errorf("failed to GetDir %+v: %v", meta, err)
				}
			}
			fullHash.Write([]byte(crr.Hash))
			rr.Add(crr)
		}
	}
	// TODO(tsileo) sum the hash and check with the root
	rr.DirsCount++
	rr.DirsDownloaded++
	rr.Hash = fmt.Sprintf("%x", fullHash.Sum(nil))
	return
}
