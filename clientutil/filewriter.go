package clientutil

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/dchest/blake2b"

	"github.com/tsileo/blobsnap/chunker"
)

var (
	MinBlobSize = 64 << 10 // 64Kb
	MaxBlobSize = 1 << 20  // 1MB
)

// FileWriter reads the file byte and byte and upload it,
// chunk by chunk, it also constructs the file index .
func (up *Uploader) FileWriter(key, path string, meta *Meta) (*WriteResult, error) {
	writeResult := NewWriteResult()
	// Init the rolling checksum
	rs := chunker.New()
	// Open the file
	f, err := os.Open(path)
	defer f.Close()
	if err != nil {
		return writeResult, fmt.Errorf("can't open file %v: %v", path, err)
	}
	// Prepare the reader to compute the hash on the fly
	fullHash := blake2b.New256()
	freader := io.TeeReader(f, fullHash)
	eof := false
	i := 0
	// TODO don't read one byte at a time if meta.Size < chunker.ChunkMinSize
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
		//if (onSplit && (buf.Len() > MinBlobSize)) || buf.Len() >= MaxBlobSize || eof {
		if onSplit || eof {
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
			meta.AddIndexedRef(writeResult.Size, nsha)
			//tx.Ladd(key, writeResult.Size, nsha)
			rs.Reset()
		}
		if eof {
			break
		}
	}
	writeResult.Hash = fmt.Sprintf("%x", fullHash.Sum(nil))
	if writeResult.BlobsUploaded > 0 {
		writeResult.FilesCount++
		writeResult.FilesUploaded++
	}
	return writeResult, nil
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
	meta := NewMeta()
	meta.Name = filename
	meta.Size = int(fstat.Size())
	meta.Type = "file"
	meta.ModTime = fstat.ModTime().Format(time.RFC3339)
	meta.Mode = uint32(fstat.Mode())
	wr := NewWriteResult()
	if fstat.Size() > 0 {
		cwr, err := up.FileWriter(sha, path, meta)
		if err != nil {
			return nil, nil, fmt.Errorf("FileWriter error: %v", err)
		}
		wr.free()
		wr = cwr
	}
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
