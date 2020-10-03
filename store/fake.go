package store

type FakeBlobStore struct {
}

func (bs FakeBlobStore) Stat(hash string) (bool, error) {
    return false, nil
}

func (bs FakeBlobStore) Get(hash string) ([]byte, error) {
    return nil, nil
}

func (bs FakeBlobStore) Put(hash string, data []byte) error {
    return nil
}

type FakeKvStore struct {
}

func (kvs FakeKvStore) Put(key, data string, ver int64) error {
    return nil
}

func (kvs FakeKvStore) Entries(begin, end string, limit int) ([]*Entry, error) {
    return nil, nil
}

func (kvs FakeKvStore) Versions(key string, begin, end int64, limit int) (*EntryVersions, error) {
    return nil, nil
}
