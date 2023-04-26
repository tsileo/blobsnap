package store

import (
	"bytes"
	"context"
	"fmt"
	"net/http"
	"path"

	yadisk "github.com/antonovvk/yandex-disk-sdk-go"
	log "github.com/inconshreveable/log15"
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
	logger := log.New("hash", hash)
	logger.Debug("YaDisk stat")

	_, err := yd.yaDisk.GetResource(yd.fileName(hash), nil, 0, 0, false, "", "")
	if e, _ := err.(*yadisk.Error); e != nil && e.ErrorID == "DiskNotFoundError" {
		return false, nil
	} else if err != nil {
		logger.Error("YaDisk resource error", "error", err)
		return false, nil
	}
	return true, nil
}

func (yd YaDiskBlobStore) Get(hash string) ([]byte, error) {
	if yd.cache != nil {
		if data, err := yd.cache.Get(hash); err == nil {
			return data, nil
		}
	}
	logger := log.New("hash", hash)
	logger.Debug("YaDisk download")

	link, err := yd.yaDisk.GetResourceDownloadLink(yd.fileName(hash), nil)
	if err != nil {
		logger.Error("YaDisk resource error", "error", err)
		return nil, err
	}

	data, err := yd.yaDisk.PerformDownload(link)
	if err != nil {
		logger.Error("YaDisk resource download error", "error", err)
		return nil, err
	}

	if yd.cache != nil {
		if err := yd.cache.Put(hash, data); err != nil {
			logger.Warn("YaDisk cache error", "error", err)
		}
	}

	logger.Debug("YaDisk download done")
	return data, nil
}

func (yd YaDiskBlobStore) Put(hash string, data []byte) (err error) {
	logger := log.New("hash", hash)
	logger.Debug("YaDisk upload")
	link, err := yd.yaDisk.GetResourceUploadLink(yd.fileName(hash), nil, false)
	if err != nil {
		return err
	}
	if _, err := yd.yaDisk.PerformUpload(link, bytes.NewBuffer(data)); err != nil {
		logger.Error("YaDisk upload error", "error", err)
		return err
	}

	status, err := yd.yaDisk.GetOperationStatus(link.OperationID, nil)
	if err != nil {
		logger.Error("YaDisk upload status error", "error", err)
		return err
	}
	if status.Status != "success" {
		logger.Error("YaDisk upload failed", "status", status.Status)
		return fmt.Errorf("Upload failed: %s", status.Status)
	}

	logger.Debug("YaDisk upload done")
	return nil
}

func (yd YaDiskBlobStore) Close() {
	return
}
