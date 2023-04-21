package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sort"

	bolt "go.etcd.io/bbolt"
)

type BoltKvStore struct {
	db *bolt.DB
}

func NewBoltKbStore(dbFileName string) (*BoltKvStore, error) {
	db, err := bolt.Open(dbFileName, 0600, nil)
	if err != nil {
		return nil, err
	}
	for _, bucket := range []string{"entries"} {
		if err := db.Update(func(tx *bolt.Tx) error {
			_, err := tx.CreateBucketIfNotExists([]byte(bucket))
			return err
		}); err != nil {
			return nil, fmt.Errorf("Failed to create bucket %s: %v", bucket, err)
		}
	}
	return &BoltKvStore{db}, nil
}

func (kvs BoltKvStore) Put(key string, data []byte, ver int64) error {
	return kvs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("entries"))

		var ver int64
		v := b.Get([]byte(key))
		ent := Entries{}
		if err := ent.decodeOne(key, bytes.NewReader(v)); err != nil {
			return err
		} else if len(ent) != 0 {
			ver = ent[0].Version
		}
		ver++
		buf := &bytes.Buffer{}
		encodeOne(buf, ver, data)
		buf.Write(v)
		return b.Put([]byte(key), buf.Bytes())
	})
}

func (kvs BoltKvStore) Dump() (res Entries, err error) {
	err = kvs.db.View(func(tx *bolt.Tx) error {
		return tx.Bucket([]byte("entries")).ForEach(func(k, v []byte) error {
			return res.decodeAll(string(k), bytes.NewReader(v))
		})
	})
	return res, err
}

func (kvs BoltKvStore) Entries(begin, end string, limit int) (res Entries, err error) {
	err = kvs.db.View(func(tx *bolt.Tx) error {
		c := tx.Bucket([]byte("entries")).Cursor()
		for k, v := c.Seek([]byte(begin)); k != nil && bytes.Compare(k, []byte(end)) <= 0; k, v = c.Next() {
			if err := res.decodeOne(string(k), bytes.NewReader(v)); err != nil {
				return err
			}
		}
		return nil
	})
	return res, err
}

func (kvs BoltKvStore) Versions(key string, begin, end int64, limit int) (res *EntryVersions, err error) {
	err = kvs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("entries"))

		v := b.Get([]byte(key))
		vers := Entries{}
		if err := vers.decodeAll(key, bytes.NewReader(v)); err != nil {
			return err
		}
		res = &EntryVersions{Key: key}
		for _, v := range vers {
			if v.Version >= end {
				continue
			} else if v.Version < begin {
				break
			}
			res.Versions = append(res.Versions, v)
		}
		sort.Slice(res.Versions, func(i, j int) bool { return res.Versions[i].Version < res.Versions[j].Version })
		return nil
	})
	return res, err
}

func (kvs BoltKvStore) Close() {
	if kvs.db != nil {
		kvs.db.Close()
	}
}

func (es *Entries) decodeAll(key string, r io.Reader) error {
	for {
		n := len(*es)
		if err := es.decodeOne(key, r); err != nil {
			return err
		} else if len(*es) == n {
			break
		}
	}
	return nil
}

type header struct {
	Ver int64
	Len int64
}

func (es *Entries) decodeOne(key string, r io.Reader) error {
	h := header{}
	if err := binary.Read(r, binary.LittleEndian, &h); err != nil {
		if err == io.EOF {
			err = nil
		}
		return err
	}
	data := make([]byte, h.Len)
	if _, err := io.ReadFull(r, data); err != nil {
		return fmt.Errorf("Failed to read data for key: %s ver: %d error: %v", key, h.Ver, err)
	}
	*es = append(*es, &Entry{Version: h.Ver, Key: key, Data: data})
	return nil
}

func encodeOne(w io.Writer, ver int64, data []byte) {
	binary.Write(w, binary.LittleEndian, header{ver, int64(len(data))})
	w.Write(data)
}
