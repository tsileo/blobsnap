package clientutil

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/dchest/blake2b"

	"github.com/tsileo/blobsnap/rolling"
	"github.com/tsileo/blobstash/client"
)

var (
	MinBlobSize = 64 << 10 // 64Kb
	MaxBlobSize = 1 << 20  // 1MB
)

// FileWriter reads the file byte and byte and upload it,
// chunk by chunk, it also constructs the file index .
func (up *Uploader) FileWriter(key, path string) (string, *WriteResult, error) {
	metaContent := NewMetaContent()
	writeResult := NewWriteResult()
	// Init the rolling checksum
	window := 64
	rs := rolling.New(window)
	// Open the file
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		return "", writeResult, fmt.Errorf("can't open file %v: %v", path, err)
	}
	// Prepare the reader to compute the hash on the fly
	fullHash := blake2b.New256()
	freader := io.TeeReader(f, fullHash)
	eof := false
	i := 0
	// Prepare the blob writer
	var buf bytes.Buffer
	blobHash := blake2b.New256()
	blobWriter := io.MultiWriter(&buf, blobHash, rs)
	for {
		b := make([]byte, 1)
		_, err := freader.Read(b)
		if err == io.EOF {
			eof = true
		} else {
			blobWriter.Write(b)
			i++
		}
		onSplit := rs.OnSplit()
		if (onSplit && (buf.Len() > MinBlobSize)) || buf.Len() >= MaxBlobSize || eof {
			nsha := fmt.Sprintf("%x", blobHash.Sum(nil))
			// Check if the blob exists
			exists, err := up.bs.Stat(nsha)
			if err != nil {
				panic(fmt.Sprintf("DB error: %v", err))
			}
			if !exists {
				if err := up.bs.Put(nsha, buf.Bytes()); err != nil {
					panic(fmt.Errorf("failed to PUT blob %v", err))
				}
				writeResult.BlobsUploaded++
				writeResult.SizeUploaded += buf.Len()
			} else {
				writeResult.SizeSkipped += buf.Len()
				writeResult.BlobsSkipped++
			}
			writeResult.Size += buf.Len()
			buf.Reset()
			blobHash.Reset()
			writeResult.BlobsCount++
			// Save the location and the blob hash into a sorted list (with the offset as index)
			metaContent.Add(writeResult.Size, nsha)
			//tx.Ladd(key, writeResult.Size, nsha)
		}
		if eof {
			break
		}
	}
	writeResult.Hash = fmt.Sprintf("%x", fullHash.Sum(nil))
	writeResult.FilesCount++
	writeResult.FilesUploaded++
	mhash, mjs := metaContent.Json()
	if up.bs.Put(mhash, mjs); err != nil {
		return "", nil, err
	}
	writeResult.BlobsCount++
	writeResult.BlobsUploaded++
	writeResult.Size += len(mjs)
	writeResult.SizeUploaded += len(mjs)
	// TODO where to store mhash ? vkv ?
	return mhash, writeResult, nil
}

func (up *Uploader) PutFile(path string) (*Meta, *WriteResult, error) {
	up.StartUpload()
	defer up.UploadDone()
	fstat, err := os.Stat(path)
	if os.IsNotExist(err) {
		return nil, nil, err
	}
	_, filename := filepath.Split(path)
	sha, err := FullHash(path)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to compute fulle hash %v: %v", path, err)
	}
	// First we check if the file isn't already uploaded,
	// if so we skip it.

	// TODO use vkv to check the file: full hash => mete content hash

	//exists, err := up.bs.Stat(sha)
	//if err != nil {
	//	return nil, nil, fmt.Errorf("failed to stat %v: %v", sha, err)
	//}
	metaRef := ""
	kv, err := up.kvs.Get(fmt.Sprintf("blobsnap:map:%v", sha), -1)
	exists := false
	if err != nil {
		if err != client.ErrKeyNotFound {
			return nil, nil, fmt.Errorf("failed to query blobsnap:map : %v", err)
		}
	} else {
		metaRef = kv.Value
		exists = true
	}
	wr := NewWriteResult()
	if exists || fstat.Size() == 0 {
		wr.Hash = sha
		wr.AlreadyExists = true
		wr.FilesSkipped++
		wr.FilesCount++
		wr.SizeSkipped = int(fstat.Size())
		wr.Size = wr.SizeSkipped
		//	wr.BlobsCount += cnt
		//		wr.BlobsSkipped += cnt
	} else {
		mref, cwr, err := up.FileWriter(sha, path)
		if err != nil {
			return nil, nil, fmt.Errorf("FileWriter error: %v", err)
		}
		wr.free()
		wr = cwr
		metaRef = mref
		_, err = up.kvs.Put(fmt.Sprintf("blobsnap:map:%v", sha), mref, -1)
		if err != nil {
			return nil, nil, fmt.Errorf("failed to update blobsnap:map : %v", err)
		}
	}
	meta := NewMeta()
	meta.Ref = metaRef
	meta.Name = filename
	meta.Size = int(fstat.Size())
	meta.Type = "file"
	meta.ModTime = fstat.ModTime().Format(time.RFC3339)
	meta.Mode = uint32(fstat.Mode())
	//meta.ComputeHash()
	mhash, mjs := meta.Json()
	mexists, err := up.bs.Stat(mhash)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to stat blob %v: %v", mhash, err)
	}
	wr.Size += len(mjs)
	if !mexists {
		if err := up.bs.Put(mhash, mjs); err != nil {
			return nil, nil, fmt.Errorf("failed to put blob %v: %v", mhash, err)
		}
		wr.BlobsCount++
		wr.BlobsUploaded++
		wr.SizeUploaded += len(mjs)
	} else {
		wr.SizeSkipped += len(mjs)
	}
	meta.Hash = mhash
	return meta, wr, nil
}
