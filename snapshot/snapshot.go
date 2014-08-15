package snapshot

import (
	"fmt"

	"github.com/dchest/blake2b"
)

type SnapSet struct {
	Path    string `redis:"path"`
	Hostname string `redis:"hostname"`
	Hash    string `redis:"-"`
}

// SetKey returns the snapshot "set key",
// computed by hashing:
// Abosolute path + hostname
func SetKey(path, hostname string) string {
	hash := blake2b.New256()
	hash.Write([]byte(path))
	hash.Write([]byte(hostname))
	return fmt.Sprintf("%x", hash.Sum(nil))
}
