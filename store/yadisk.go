package store

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path"

	log "github.com/inconshreveable/log15"
	yadisk "github.com/antonovvk/yandex-disk-sdk-go"
)

type YaDiskBlobStore struct {
	root   string
	yaDisk yadisk.YaDisk
	cache  *FileBlobStore
}

func NewYaDiskBlobStore(token, root string) (*YaDiskBlobStore, error) {
	yaDisk, err := yadisk.NewYaDisk(context.Background(), http.DefaultClient, &yadisk.Token{AccessToken: token})
	if err != nil {
		return nil, err
	}
	return &YaDiskBlobStore{root, yaDisk, nil}, nil
}

func (yd *YaDiskBlobStore) WithCache(dir string) *YaDiskBlobStore {
	res := *yd
	res.cache = &FileBlobStore{dir}
	return &res
}

func (yd YaDiskBlobStore) fileName(hash string) string {
	return path.Join(yd.root, hash)
}

func (yd YaDiskBlobStore) Stat(hash string) (bool, error) {
	if yd.cache != nil {
		if ok, _ := yd.cache.Stat(hash); ok {
			return true, nil
		}
	}
	link, err := yd.yaDisk.GetResourceDownloadLink(yd.fileName(hash), nil)
	if err != nil {
		log.Error("YaDisk resource error", "hash", hash, "error", err)
		return false, nil
	}
	log.Debug("YaDisk stat", "link", link.Href)
	return true, nil
}

func (yd YaDiskBlobStore) Get(hash string) ([]byte, error) {
	if yd.cache != nil {
		if data, err := yd.cache.Get(hash); err == nil {
			return data, nil
		}
	}

	link, err := yd.yaDisk.GetResourceDownloadLink(yd.fileName(hash), nil)
	if err != nil {
		log.Error("YaDisk resource error", "hash", hash, "error", err)
		return nil, err
	}

	log.Debug("YaDisk download", "link", link.Href)

	data, err := yd.yaDisk.PerformDownload(link)
	if err != nil {
		log.Error("YaDisk resource download error", "hash", hash, "error", err)
		return nil, err
	}

	if yd.cache != nil {
		if err := yd.cache.Put(hash, data); err != nil {
			log.Warn("YaDisk cache error", "hash", hash, "error", err)
		}
	}
	return data, nil
}

func (yd YaDiskBlobStore) Put(hash string, data []byte) (err error) {
	link, err := yd.yaDisk.GetResourceUploadLink(yd.fileName(hash), nil, false)
	if err != nil {
		return err
	}
	if _, err := yd.yaDisk.PerformUpload(link, bytes.NewBuffer(data)); err != nil {
		log.Error("YaDisk upload error", "hash", hash, "error", err)
		return err
	}

	status, err := yd.yaDisk.GetOperationStatus(link.OperationID, nil)
	if err != nil {
		log.Error("YaDisk upload status error", "hash", hash, "error", err)
		return err
	}
	if status.Status != "success" {
		log.Error("YaDisk upload failed", "hash", hash, "status", status.Status)
		return fmt.Errorf("Upload failed: %s", status.Status)
	}
	return nil
}

func (yd YaDiskBlobStore) Close() {
	return
}
