package store

type BlobStore interface {
	Stat(hash string) (bool, error)
	Get(hash string) ([]byte, error)
	Put(hash string, data []byte) error
	Close()
}

type Entry struct {
	Version int64  `json:"ver"`
	Key     string `json:"key,omitempty"`
	Data    []byte `json:"data,omitempty"`
}

type Entries []*Entry

type EntryVersions struct {
	Key      string   `json:"key"`
	Versions []*Entry `json:"versions"`
}

type KvStore interface {
	Dump() (Entries, error)
	Put(key string, data []byte, ver int64) error
	Entries(begin, end string, limit int) (Entries, error)
	Versions(key string, begin, end int64, limit int) (*EntryVersions, error)
	Close()
}
