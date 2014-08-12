package snapshot

import (
	"fmt"
	"strconv"
	"time"

	"github.com/dchest/blake2b"
)

type Snapshot struct {
	Path    string `redis:"path"`
	Hostname string `redis:"hostname"`
	MetaRef    string `redis:"meta_ref"`
	Time    int64    `redis:"time"`
	Hash    string `redis:"-"`
}


type SnapSet struct {
	Path    string `redis:"path"`
	Hostname string `redis:"hostname"`
	Hash    string `redis:"-"`
}

// SetKey returns the snapshot "set key",
// computed by hashing:
// Abosolute path + hostname
func (s *Snapshot) SetKey() string {
	hash := blake2b.New256()
	hash.Write([]byte(s.Path))
	hash.Write([]byte(s.Hostname))
	return fmt.Sprintf("%x", hash.Sum(nil))
}

func NewSnapshot(path, hostname, metaRef string) *Snapshot {
	return &Snapshot{
		Time: time.Now().UTC().Unix(),
		Path: path,
		Hostname: hostname,
		MetaRef: metaRef,
	}
}

func (s *Snapshot) ComputeHash() {
	hash := blake2b.New256()
	hash.Write([]byte(s.Path))
	hash.Write([]byte(s.Hostname))
	hash.Write([]byte(strconv.Itoa(int(s.Time))))
	s.Hash = fmt.Sprintf("%x", hash.Sum(nil))
	return
}
