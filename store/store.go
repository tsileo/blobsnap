package store

type BlobStore interface {
	Stat(hash string) (bool, error)
	Get(hash string) ([]byte, error)
	Put(hash string, data []byte) error
}

type Entry struct {
	Version int64  `json:"ver"`
	Key     string `json:"key,omitempty"`
	Hash    string `json:"hash,omitempty"`
	Data    string `json:"data,omitempty"`
}

type EntryVersions struct {
	Key      string   `json:"key"`
	Versions []*Entry `json:"versions"`
}

type KvStore interface {
	Put(key, data string, ver int64) error
	Entries(begin, end string, limit int) ([]*Entry, error)
	Versions(key string, begin, end int64, limit int) (*EntryVersions, error)
}
