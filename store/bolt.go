package store

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"sort"

	"github.com/boltdb/bolt"
)

type BoltKvStore struct {
	db *bolt.DB
}

func NewBoltKbStore(dbFileName string) (*BoltKvStore, error) {
	db, err := bolt.Open(dbFileName, 0600, nil)
	if err != nil {
		return nil, err
	}
	return &BoltKvStore{db}, nil
}

func (kvs BoltKvStore) Put(key string, data []byte, ver int64) error {
	return kvs.db.Update(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("entries"))

		var ver int64
		v := b.Get([]byte(key))
		if ent, err := decodeOne(key, bytes.NewReader(v)); err != nil {
			return err
		} else if ent != nil {
			ver = ent.Version
		}
		ver++
		buf := &bytes.Buffer{}
		encodeOne(buf, ver, data)
		buf.Write(v)
		return b.Put([]byte(key), buf.Bytes())
	})
}

func (kvs BoltKvStore) Entries(begin, end string, limit int) (res []*Entry, err error) {
	err = kvs.db.View(func(tx *bolt.Tx) error {
		b := tx.Bucket([]byte("entries"))

		c := b.Cursor()
		for k, v := c.Seek([]byte(begin)); k != nil && bytes.Compare(k, []byte(end)) <= 0; k, v = c.Next() {
			if ent, err := decodeOne(string(k), bytes.NewReader(v)); err != nil {
				return err
			} else if ent != nil {
				res = append(res, ent)
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
		vers, err := decodeAll(key, bytes.NewReader(v))
		if err != nil {
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

func decodeAll(key string, r io.Reader) (res []*Entry, err error) {
	for {
		if ent, err := decodeOne(key, r); err != nil {
			return nil, err
		} else if ent != nil {
			res = append(res, ent)
		} else {
			break
		}
	}
	return res, nil
}

type header struct {
	Ver int64
	Len int
}

func decodeOne(key string, r io.Reader) (res *Entry, err error) {
	h := header{}
	if err := binary.Read(r, binary.LittleEndian, &h); err != nil {
		if err == io.EOF {
			err = nil
		}
		return nil, err
	}
	data := make([]byte, h.Len)
	if _, err := io.ReadFull(r, data); err != nil {
		return nil, fmt.Errorf("Failed to read data for key: %s ver: %d error: %v", key, h.Ver, err)
	}
	return &Entry{Version: h.Ver, Key: key, Data: data}, nil
}

func encodeOne(w io.Writer, ver int64, data []byte) {
	binary.Write(w, binary.LittleEndian, header{ver, len(data)})
	w.Write(data)
}