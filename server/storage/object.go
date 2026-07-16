package storage

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
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

type Store interface {
	Put(ctx context.Context, key string, data io.Reader, size int64, contentType string) error
	Open(ctx context.Context, key string) (io.ReadCloser, string, error)
	OpenRange(ctx context.Context, key string, offset int64, size int64) (io.ReadCloser, error)
	List(ctx context.Context, prefix string) ([]Info, error)
	Delete(ctx context.Context, key string) error
}

type Info struct {
	Key       string
	Size      int64
	ETag      string
	UpdatedAt time.Time
}

var (
	s3StoreCacheMu sync.Mutex
	s3StoreCache   = map[string]Store{}
)

func NewFromEnv() (Store, error) {
	storageURL := storageURL()
	if strings.HasPrefix(storageURL, "http://") || strings.HasPrefix(storageURL, "https://") {
		return newS3Store(storageURL)
	}
	return localStore{root: Root()}, nil
}

func newS3Store(storageURL string) (Store, error) {
	s3StoreCacheMu.Lock()
	defer s3StoreCacheMu.Unlock()
	if store, ok := s3StoreCache[storageURL]; ok {
		return store, nil
	}
	config, err := parseS3Storage(storageURL)
	if err != nil {
		return nil, err
	}
	client, err := minio.New(config.endpoint, &minio.Options{
		Creds:        credentials.NewStaticV4(config.access, config.secret, ""),
		Secure:       config.secure,
		BucketLookup: config.lookup,
	})
	if err != nil {
		return nil, err
	}
	store := s3Store{client: client, bucket: config.bucket}
	s3StoreCache[storageURL] = store
	return store, nil
}

func Root() string {
	return storageURL()
}

func storageURL() string {
	if value := strings.TrimSpace(os.Getenv("STORAGE")); value != "" {
		return value
	}
	return defaultStorageRoot()
}

func defaultStorageRoot() string {
	home, err := os.UserHomeDir()
	if err == nil && strings.TrimSpace(home) != "" {
		return home
	}
	return "storage"
}

type s3StorageConfig struct {
	endpoint string
	access   string
	secret   string
	bucket   string
	secure   bool
	lookup   minio.BucketLookupType
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
	query := parsed.Query()
	lookup, err := parseBucketLookup(query.Get("lookup"))
	if err != nil {
		return s3StorageConfig{}, err
	}
	return s3StorageConfig{
		endpoint: parsed.Host,
		access:   access,
		secret:   secret,
		bucket:   bucket,
		secure:   parsed.Scheme == "https",
		lookup:   lookup,
	}, nil
}

func parseBucketLookup(value string) (minio.BucketLookupType, error) {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "auto":
		return minio.BucketLookupAuto, nil
	case "dns":
		return minio.BucketLookupDNS, nil
	case "path":
		return minio.BucketLookupPath, nil
	default:
		return minio.BucketLookupAuto, fmt.Errorf("STORAGE lookup must be auto, dns, or path")
	}
}

func CleanKey(raw string) (string, error) {
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

func IsNotFound(err error) bool {
	if os.IsNotExist(err) {
		return true
	}
	switch minio.ToErrorResponse(err).Code {
	case "NoSuchBucket", "NoSuchKey", "NoSuchObject", "NotFound":
		return true
	default:
		return false
	}
}

func cleanListPrefix(raw string) (string, error) {
	key, err := CleanKey(raw)
	if err != nil {
		return "", err
	}
	return key + "/", nil
}

type localStore struct {
	root string
}

func (store localStore) Put(_ context.Context, key string, data io.Reader, _ int64, _ string) error {
	clean, err := CleanKey(key)
	if err != nil {
		return err
	}
	filePath := filepath.Join(store.root, filepath.FromSlash(clean))
	if err := os.MkdirAll(filepath.Dir(filePath), 0o700); err != nil {
		return err
	}
	if err := os.Chmod(filepath.Dir(filePath), 0o700); err != nil {
		return err
	}
	file, err := os.CreateTemp(filepath.Dir(filePath), ".object-*")
	if err != nil {
		return err
	}
	tmp := file.Name()
	defer os.Remove(tmp)
	if err := file.Chmod(0o600); err != nil {
		_ = file.Close()
		return err
	}
	if _, err := io.Copy(file, data); err != nil {
		_ = file.Close()
		return err
	}
	if err := file.Close(); err != nil {
		return err
	}
	return os.Rename(tmp, filePath)
}

func (store localStore) Open(_ context.Context, key string) (io.ReadCloser, string, error) {
	clean, err := CleanKey(key)
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

func (store localStore) OpenRange(_ context.Context, key string, offset int64, size int64) (io.ReadCloser, error) {
	if offset < 0 || size <= 0 {
		return nil, fmt.Errorf("invalid object range")
	}
	clean, err := CleanKey(key)
	if err != nil {
		return nil, err
	}
	file, err := os.Open(filepath.Join(store.root, filepath.FromSlash(clean)))
	if err != nil {
		return nil, err
	}
	info, err := file.Stat()
	if err != nil {
		_ = file.Close()
		return nil, err
	}
	if offset > info.Size() || size > info.Size()-offset {
		_ = file.Close()
		return nil, fmt.Errorf("object range exceeds size")
	}
	return struct {
		io.Reader
		io.Closer
	}{Reader: io.NewSectionReader(file, offset, size), Closer: file}, nil
}

func (store localStore) List(_ context.Context, prefix string) ([]Info, error) {
	clean, err := CleanKey(prefix)
	if err != nil {
		return nil, err
	}
	root := filepath.Join(store.root, filepath.FromSlash(clean))
	items := []Info{}
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
		items = append(items, Info{Key: filepath.ToSlash(relative), Size: info.Size(), UpdatedAt: info.ModTime()})
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
	clean, err := CleanKey(key)
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
}

func (store s3Store) Put(ctx context.Context, key string, data io.Reader, size int64, contentType string) error {
	clean, err := CleanKey(key)
	if err != nil {
		return err
	}
	_, err = store.client.PutObject(ctx, store.bucket, clean, data, size, minio.PutObjectOptions{ContentType: contentType})
	return err
}

func (store s3Store) Open(ctx context.Context, key string) (io.ReadCloser, string, error) {
	clean, err := CleanKey(key)
	if err != nil {
		return nil, "", err
	}
	object, info, headers, err := (minio.Core{Client: store.client}).GetObject(ctx, store.bucket, clean, minio.GetObjectOptions{})
	if err != nil {
		return nil, "", err
	}
	contentType := info.ContentType
	if contentType == "" {
		contentType = headers.Get("Content-Type")
	}
	if contentType == "" {
		contentType = mime.TypeByExtension(filepath.Ext(clean))
	}
	return object, contentType, nil
}

func (store s3Store) OpenRange(ctx context.Context, key string, offset int64, size int64) (io.ReadCloser, error) {
	if offset < 0 || size <= 0 {
		return nil, fmt.Errorf("invalid object range")
	}
	clean, err := CleanKey(key)
	if err != nil {
		return nil, err
	}
	opts := minio.GetObjectOptions{}
	if err := opts.SetRange(offset, offset+size-1); err != nil {
		return nil, err
	}
	reader, _, headers, err := (minio.Core{Client: store.client}).GetObject(ctx, store.bucket, clean, opts)
	if err != nil {
		return nil, err
	}
	want := fmt.Sprintf("bytes %d-%d/", offset, offset+size-1)
	if !strings.HasPrefix(headers.Get("Content-Range"), want) || headers.Get("Content-Length") != strconv.FormatInt(size, 10) {
		_ = reader.Close()
		return nil, fmt.Errorf("S3 backend ignored object range")
	}
	return reader, nil
}

func (store s3Store) List(ctx context.Context, prefix string) ([]Info, error) {
	clean, err := cleanListPrefix(prefix)
	if err != nil {
		return nil, err
	}
	items := []Info{}
	for object := range store.client.ListObjects(ctx, store.bucket, minio.ListObjectsOptions{Prefix: clean, Recursive: true}) {
		if object.Err != nil {
			if response := minio.ToErrorResponse(object.Err); response.Code == "NoSuchBucket" {
				return items, nil
			}
			return nil, object.Err
		}
		items = append(items, Info{Key: object.Key, Size: object.Size, ETag: object.ETag, UpdatedAt: object.LastModified})
	}
	return items, nil
}

func (store s3Store) Delete(ctx context.Context, key string) error {
	clean, err := CleanKey(key)
	if err != nil {
		return err
	}
	return store.client.RemoveObject(ctx, store.bucket, clean, minio.RemoveObjectOptions{})
}
