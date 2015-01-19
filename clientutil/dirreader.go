package clientutil

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/dchest/blake2b"

	"github.com/tsileo/blobstash/client2"
)

// GetDir restore the directory to path
func GetDir(bs *client2.BlobStore, key, path string) (rr *ReadResult, err error) {
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
		metacontent, err := meta.FetchMetaContent(bs)
		if err != nil {
			return nil, fmt.Errorf("failed to fetch meta content: %v", err)
		}
		for _, hash := range metacontent.Mapping {
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
