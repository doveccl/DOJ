package utils

import (
	"context"
	"errors"
	"fmt"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type ObjectStore interface {
	Put(ctx context.Context, key string, data io.Reader, size int64, contentType string) error
	Open(ctx context.Context, key string) (io.ReadCloser, string, error)
	List(ctx context.Context, prefix string) ([]ObjectInfo, error)
	Delete(ctx context.Context, key string) error
}

type ObjectInfo struct {
	Key  string
	Size int64
}

const defaultStorageRoot = "/var/lib/doj"

func NewObjectStoreFromEnv() (ObjectStore, error) {
	storage := storageURL()
	if strings.HasPrefix(storage, "http://") || strings.HasPrefix(storage, "https://") {
		config, err := parseS3Storage(storage)
		if err != nil {
			return nil, err
		}
		client, err := minio.New(config.endpoint, &minio.Options{
			Creds:  credentials.NewStaticV4(config.access, config.secret, ""),
			Secure: config.secure,
			Region: config.region,
		})
		if err != nil {
			return nil, err
		}
		return s3Store{client: client, bucket: config.bucket, region: config.region}, nil
	}
	return localStore{root: UploadRoot()}, nil
}

func UploadRoot() string {
	return storageURL()
}

func storageURL() string {
	if value := strings.TrimSpace(os.Getenv("STORAGE")); value != "" {
		return value
	}
	return defaultStorageRoot
}

type s3StorageConfig struct {
	endpoint string
	access   string
	secret   string
	bucket   string
	region   string
	secure   bool
}

func parseS3Storage(raw string) (s3StorageConfig, error) {
	parsed, err := url.Parse(raw)
	if err != nil {
		return s3StorageConfig{}, err
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return s3StorageConfig{}, fmt.Errorf("STORAGE must start with http:// or https:// for S3-compatible storage")
	}
	if parsed.Host == "" {
		return s3StorageConfig{}, fmt.Errorf("STORAGE S3 endpoint host is required")
	}
	access := parsed.User.Username()
	secret, _ := parsed.User.Password()
	if access == "" || secret == "" {
		return s3StorageConfig{}, fmt.Errorf("STORAGE S3 credentials are required")
	}
	bucket := strings.Trim(parsed.Path, "/")
	if bucket == "" || strings.Contains(bucket, "/") {
		return s3StorageConfig{}, fmt.Errorf("STORAGE S3 URL path must be exactly one bucket name")
	}
	return s3StorageConfig{
		endpoint: parsed.Host,
		access:   access,
		secret:   secret,
		bucket:   bucket,
		region:   parsed.Query().Get("region"),
		secure:   parsed.Scheme == "https",
	}, nil
}

func CleanObjectKey(raw string) (string, error) {
	normalized := strings.TrimSpace(strings.ReplaceAll(raw, "\\", "/"))
	for _, segment := range strings.Split(normalized, "/") {
		if segment == ".." {
			return "", errors.New("invalid object key")
		}
	}
	key := path.Clean("/" + normalized)
	key = strings.TrimPrefix(key, "/")
	if key == "" || key == "." || strings.HasPrefix(key, "../") {
		return "", errors.New("invalid object key")
	}
	return key, nil
}

type localStore struct {
	root string
}

func (store localStore) Put(_ context.Context, key string, data io.Reader, _ int64, _ string) error {
	clean, err := CleanObjectKey(key)
	if err != nil {
		return err
	}
	filePath := filepath.Join(store.root, filepath.FromSlash(clean))
	if err := os.MkdirAll(filepath.Dir(filePath), 0o755); err != nil {
		return err
	}
	file, err := os.Create(filePath)
	if err != nil {
		return err
	}
	defer file.Close()
	_, err = io.Copy(file, data)
	return err
}

func (store localStore) Open(_ context.Context, key string) (io.ReadCloser, string, error) {
	clean, err := CleanObjectKey(key)
	if err != nil {
		return nil, "", err
	}
	file, err := os.Open(filepath.Join(store.root, filepath.FromSlash(clean)))
	if err != nil {
		return nil, "", err
	}
	contentType := mime.TypeByExtension(filepath.Ext(clean))
	if contentType == "" {
		buffer := make([]byte, 512)
		n, _ := file.Read(buffer)
		contentType = http.DetectContentType(buffer[:n])
		if _, err := file.Seek(0, io.SeekStart); err != nil {
			_ = file.Close()
			return nil, "", err
		}
	}
	return file, contentType, nil
}

func (store localStore) List(_ context.Context, prefix string) ([]ObjectInfo, error) {
	clean, err := CleanObjectKey(prefix)
	if err != nil {
		return nil, err
	}
	root := filepath.Join(store.root, filepath.FromSlash(clean))
	items := []ObjectInfo{}
	err = filepath.WalkDir(root, func(filePath string, entry os.DirEntry, err error) error {
		if err != nil {
			if os.IsNotExist(err) {
				return nil
			}
			return err
		}
		if entry.IsDir() {
			return nil
		}
		info, err := entry.Info()
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(store.root, filePath)
		if err != nil {
			return err
		}
		items = append(items, ObjectInfo{Key: filepath.ToSlash(relative), Size: info.Size()})
		return nil
	})
	if err != nil {
		if os.IsNotExist(err) {
			return items, nil
		}
		return nil, err
	}
	return items, nil
}

func (store localStore) Delete(_ context.Context, key string) error {
	clean, err := CleanObjectKey(key)
	if err != nil {
		return err
	}
	err = os.Remove(filepath.Join(store.root, filepath.FromSlash(clean)))
	if err != nil && !os.IsNotExist(err) {
		return err
	}
	return nil
}

type s3Store struct {
	client *minio.Client
	bucket string
	region string
}

func (store s3Store) Put(ctx context.Context, key string, data io.Reader, size int64, contentType string) error {
	clean, err := CleanObjectKey(key)
	if err != nil {
		return err
	}
	if err := store.ensureBucket(ctx); err != nil {
		return err
	}
	_, err = store.client.PutObject(ctx, store.bucket, clean, data, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (store s3Store) ensureBucket(ctx context.Context) error {
	exists, err := store.client.BucketExists(ctx, store.bucket)
	if err != nil {
		return err
	}
	if exists {
		return nil
	}
	return store.client.MakeBucket(ctx, store.bucket, minio.MakeBucketOptions{Region: store.region})
}

func (store s3Store) Open(ctx context.Context, key string) (io.ReadCloser, string, error) {
	clean, err := CleanObjectKey(key)
	if err != nil {
		return nil, "", err
	}
	object, err := store.client.GetObject(ctx, store.bucket, clean, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", err
	}
	stat, err := object.Stat()
	if err != nil {
		_ = object.Close()
		return nil, "", err
	}
	contentType := stat.ContentType
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(clean))
	}
	return object, contentType, nil
}

func (store s3Store) List(ctx context.Context, prefix string) ([]ObjectInfo, error) {
	clean, err := CleanObjectKey(prefix)
	if err != nil {
		return nil, err
	}
	items := []ObjectInfo{}
	for object := range store.client.ListObjects(ctx, store.bucket, minio.ListObjectsOptions{Prefix: clean, Recursive: true}) {
		if object.Err != nil {
			if response := minio.ToErrorResponse(object.Err); response.Code == "NoSuchBucket" {
				return items, nil
			}
			return nil, object.Err
		}
		items = append(items, ObjectInfo{Key: object.Key, Size: object.Size})
	}
	return items, nil
}

func (store s3Store) Delete(ctx context.Context, key string) error {
	clean, err := CleanObjectKey(key)
	if err != nil {
		return err
	}
	return store.client.RemoveObject(ctx, store.bucket, clean, minio.RemoveObjectOptions{})
}
