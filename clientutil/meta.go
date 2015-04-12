package clientutil

import (
	"encoding/json"
	"fmt"
	"sync"

	"github.com/dchest/blake2b"
	"github.com/tsileo/blobstash/client"
)

type MetaContent struct {
	Data []interface{}
}

func NewMetaContent() *MetaContent {
	return &MetaContent{
		Data: []interface{}{},
	}
}

func (mc *MetaContent) Add(index int, hash string) {
	mc.Data = append(mc.Data, []interface{}{index, hash})
}

func (mc *MetaContent) Iter() []interface{} {
	return mc.Data
}

func (mc *MetaContent) AddHash(hash string) {
	mc.Data = append(mc.Data, hash)
}

func (mc *MetaContent) Json() (string, []byte) {
	js, err := json.Marshal(mc.Data)
	if err != nil {
		panic(err)
	}
	h := fmt.Sprintf("%x", blake2b.Sum256(js))
	return h, js
}

var metaPool = sync.Pool{
	New: func() interface{} { return &Meta{} },
}

type Meta struct {
	Name    string `json:"name"`
	Type    string `json:"type"`
	Size    int    `json:"size"`
	Mode    uint32 `json:"mode"`
	ModTime string `json:"mtime"`
	Ref     string `json:"ref"`
	Hash    string `json:"-"`
}

func (m *Meta) free() {
	m.Name = ""
	m.Type = ""
	m.Size = 0
	m.Mode = 0
	m.ModTime = ""
	m.Ref = ""
	m.Hash = ""
	metaPool.Put(m)
}

func (m *Meta) FetchMetaContent(bs *client.BlobStore) (*MetaContent, error) {
	blob, err := bs.Get(m.Ref)
	if err != nil {
		return nil, err
	}
	mc := NewMetaContent()
	if err := json.Unmarshal(blob, &mc.Data); err != nil {
		return nil, err
	}
	return mc, nil
}

func (m *Meta) Json() (string, []byte) {
	js, err := json.Marshal(m)
	if err != nil {
		panic(err)
	}
	h := fmt.Sprintf("%x", blake2b.Sum256(js))
	return h, js
}

func NewMeta() *Meta {
	return metaPool.Get().(*Meta)
}

func NewMetaFromBlobStore(bs *client.BlobStore, hash string) (*Meta, error) {
	blob, err := bs.Get(hash)
	if err != nil {
		return nil, err
	}
	meta := NewMeta()
	if err := json.Unmarshal(blob, meta); err != nil {
		return nil, err
	}
	meta.Hash = hash
	return meta, err
}

// IsFile returns true if the Meta is a file.
func (m *Meta) IsFile() bool {
	if m.Type == "file" {
		return true
	}
	return false
}

// IsDir returns true if the Meta is a directory.
func (m *Meta) IsDir() bool {
	if m.Type == "dir" {
		return true
	}
	return false
}
