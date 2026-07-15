package problem

import (
	"archive/zip"
	"compress/flate"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/doveccl/doj/contract/cases"
	"github.com/doveccl/doj/server/storage"
)

const (
	MaxPackageBytes   = 512 << 20
	MaxFileBytes      = 256 << 20
	MaxExpandedBytes  = 2 << 30
	MaxPackageEntries = 10_000
)

type Package struct {
	Hash  string `json:"hash,omitempty"`
	Size  int64  `json:"size,omitempty"`
	Files []File `json:"files,omitempty"`
	Cases []Case `json:"cases,omitempty"`
}

type File struct {
	Path           string `json:"path"`
	Size           int64  `json:"size"`
	Offset         int64  `json:"offset"`
	CompressedSize int64  `json:"compressedSize"`
	CRC32          uint32 `json:"crc32"`
}

type Case struct {
	ID     string `json:"id"`
	Input  string `json:"input"`
	Answer string `json:"answer"`
	Score  *int   `json:"score,omitempty"`
}

func (item Case) Points() int {
	if item.Score == nil {
		return 10
	}
	return *item.Score
}

func Parse(raw []byte) (Package, error) {
	if len(raw) == 0 || string(raw) == "null" || string(raw) == "{}" {
		return Package{}, nil
	}
	var item Package
	if err := json.Unmarshal(raw, &item); err != nil {
		return Package{}, err
	}
	return item, nil
}

func (item Package) JSON() ([]byte, error) {
	if item.Hash == "" {
		return []byte("{}"), nil
	}
	return json.Marshal(item)
}

func ETag(raw []byte) string {
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

func ObjectKey(problemID uint, hash string) string {
	return fmt.Sprintf("problems/%d/packages/%s.zip", problemID, hash)
}

func Build(root string, destination string, oldCases []Case) (Package, error) {
	names, err := packageFiles(root)
	if err != nil {
		return Package{}, err
	}
	if err := validateCaseFiles(names); err != nil {
		return Package{}, err
	}
	if len(names) > MaxPackageEntries {
		return Package{}, fmt.Errorf("problem package has too many entries")
	}
	file, err := os.OpenFile(destination, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
	if err != nil {
		return Package{}, err
	}
	hash := sha256.New()
	writer := zip.NewWriter(io.MultiWriter(file, hash))
	for _, name := range names {
		info, err := os.Stat(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			_ = writer.Close()
			_ = file.Close()
			return Package{}, err
		}
		if info.Size() > MaxFileBytes {
			_ = writer.Close()
			_ = file.Close()
			return Package{}, fmt.Errorf("%s exceeds file size limit", name)
		}
		header := &zip.FileHeader{Name: name, Method: zip.Deflate, Modified: time.Date(1980, 1, 1, 0, 0, 0, 0, time.UTC)}
		header.SetMode(0o600)
		dst, err := writer.CreateHeader(header)
		if err != nil {
			_ = writer.Close()
			_ = file.Close()
			return Package{}, err
		}
		src, err := os.Open(filepath.Join(root, filepath.FromSlash(name)))
		if err != nil {
			_ = writer.Close()
			_ = file.Close()
			return Package{}, err
		}
		_, copyErr := io.Copy(dst, src)
		closeErr := src.Close()
		if copyErr != nil || closeErr != nil {
			_ = writer.Close()
			_ = file.Close()
			if copyErr != nil {
				return Package{}, copyErr
			}
			return Package{}, closeErr
		}
	}
	if err := writer.Close(); err != nil {
		_ = file.Close()
		return Package{}, err
	}
	if err := file.Close(); err != nil {
		return Package{}, err
	}
	info, err := os.Stat(destination)
	if err != nil {
		return Package{}, err
	}
	if info.Size() > MaxPackageBytes {
		return Package{}, fmt.Errorf("problem package exceeds size limit")
	}
	item := Package{Hash: hex.EncodeToString(hash.Sum(nil)), Size: info.Size()}
	reader, err := zip.OpenReader(destination)
	if err != nil {
		return Package{}, err
	}
	defer reader.Close()
	for _, entry := range reader.File {
		offset, err := entry.DataOffset()
		if err != nil {
			return Package{}, err
		}
		item.Files = append(item.Files, File{
			Path: entry.Name, Size: int64(entry.UncompressedSize64), Offset: offset,
			CompressedSize: int64(entry.CompressedSize64), CRC32: entry.CRC32,
		})
	}
	item.Cases = inferCases(item.Files, oldCases)
	return item, nil
}

func validateCaseFiles(names []string) error {
	seen := map[string]string{}
	for _, name := range names {
		if !strings.HasPrefix(name, "data/") {
			continue
		}
		stem, kind := cases.DataCaseStem(strings.TrimPrefix(name, "data/"))
		if stem == "" || kind == "" {
			continue
		}
		key := stem + "\x00" + kind
		if previous := seen[key]; previous != "" {
			return fmt.Errorf("case %s has multiple %s files: %s and %s", stem, kind, previous, name)
		}
		seen[key] = name
	}
	return nil
}

func packageFiles(root string) ([]string, error) {
	var names []string
	var total int64
	for _, section := range []string{"data", "judge"} {
		base := filepath.Join(root, section)
		err := filepath.WalkDir(base, func(file string, entry os.DirEntry, err error) error {
			if os.IsNotExist(err) {
				return nil
			}
			if err != nil || entry.IsDir() {
				return err
			}
			info, err := entry.Info()
			if err != nil {
				return err
			}
			total += info.Size()
			if total > MaxExpandedBytes {
				return fmt.Errorf("problem package expands beyond limit")
			}
			rel, err := filepath.Rel(root, file)
			if err != nil {
				return err
			}
			name := filepath.ToSlash(rel)
			if clean, err := storage.CleanKey(name); err != nil || clean != name {
				return fmt.Errorf("invalid package path %q", name)
			}
			names = append(names, name)
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	sort.Strings(names)
	return names, nil
}

func inferCases(files []File, old []Case) []Case {
	inputs := map[string]string{}
	answers := map[string]string{}
	points := map[string]*int{}
	for _, item := range old {
		points[item.ID] = item.Score
	}
	for _, file := range files {
		if !strings.HasPrefix(file.Path, "data/") {
			continue
		}
		stem, kind := cases.DataCaseStem(strings.TrimPrefix(file.Path, "data/"))
		switch kind {
		case "in":
			inputs[stem] = file.Path
		case "out":
			answers[stem] = file.Path
		}
	}
	var result []Case
	for id, input := range inputs {
		if answer := answers[id]; answer != "" {
			result = append(result, Case{ID: id, Input: input, Answer: answer, Score: points[id]})
		}
	}
	sort.Slice(result, func(i, j int) bool { return cases.CaseStemLess(result[i].ID, result[j].ID) })
	return result
}

func Extract(ctx context.Context, store storage.Store, key string, item Package, root string) error {
	reader, _, err := store.Open(ctx, key)
	if err != nil {
		return err
	}
	defer reader.Close()
	allowed := make(map[string]bool, len(item.Files))
	for _, file := range item.Files {
		clean, err := storage.CleanKey(file.Path)
		if err != nil || clean != file.Path || (!strings.HasPrefix(clean, "data/") && !strings.HasPrefix(clean, "judge/")) {
			return fmt.Errorf("invalid package path %q", file.Path)
		}
		allowed[file.Path] = true
	}
	tmp, err := os.CreateTemp(root, ".download-*.zip")
	if err != nil {
		return err
	}
	tmpName := tmp.Name()
	defer os.Remove(tmpName)
	written, err := io.Copy(tmp, io.LimitReader(reader, MaxPackageBytes+1))
	if err != nil {
		_ = tmp.Close()
		return err
	}
	if written > MaxPackageBytes {
		_ = tmp.Close()
		return fmt.Errorf("problem package exceeds size limit")
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	zr, err := zip.OpenReader(tmpName)
	if err != nil {
		return err
	}
	defer zr.Close()
	for _, entry := range zr.File {
		if !allowed[entry.Name] {
			continue
		}
		target := filepath.Join(root, filepath.FromSlash(entry.Name))
		if err := os.MkdirAll(filepath.Dir(target), 0o700); err != nil {
			return err
		}
		src, err := entry.Open()
		if err != nil {
			return err
		}
		dst, err := os.OpenFile(target, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, 0o600)
		if err != nil {
			_ = src.Close()
			return err
		}
		written, copyErr := io.Copy(dst, io.LimitReader(src, MaxFileBytes+1))
		_ = src.Close()
		closeErr := dst.Close()
		if copyErr != nil || closeErr != nil || written > MaxFileBytes {
			return fmt.Errorf("invalid package file %s", entry.Name)
		}
	}
	return nil
}

func OpenFile(ctx context.Context, store storage.Store, key string, file File) (io.ReadCloser, error) {
	if file.Offset < 0 || file.CompressedSize <= 0 || file.Size < 0 || file.Size > MaxFileBytes {
		return nil, fmt.Errorf("invalid package file range")
	}
	reader, err := store.OpenRange(ctx, key, file.Offset, file.CompressedSize)
	if err != nil {
		return nil, err
	}
	inflated := flate.NewReader(reader)
	return &checkedFileReader{
		compressed: reader,
		inflated:   inflated,
		reader:     io.LimitReader(inflated, file.Size+1),
		checksum:   crc32.NewIEEE(),
		wantSize:   file.Size,
		wantCRC:    file.CRC32,
	}, nil
}

type checkedFileReader struct {
	compressed io.Closer
	inflated   io.ReadCloser
	reader     io.Reader
	checksum   hash.Hash32
	size       int64
	wantSize   int64
	wantCRC    uint32
}

func (reader *checkedFileReader) Read(buffer []byte) (int, error) {
	n, err := reader.reader.Read(buffer)
	if n > 0 {
		reader.size += int64(n)
		_, _ = reader.checksum.Write(buffer[:n])
	}
	if reader.size > reader.wantSize || err == io.EOF && (reader.size != reader.wantSize || reader.checksum.Sum32() != reader.wantCRC) {
		return n, fmt.Errorf("package file checksum mismatch")
	}
	return n, err
}

func (reader *checkedFileReader) Close() error {
	if err := reader.inflated.Close(); err != nil {
		_ = reader.compressed.Close()
		return err
	}
	return reader.compressed.Close()
}

func FindFile(item Package, name string) (File, bool) {
	name = path.Clean(strings.ReplaceAll(name, "\\", "/"))
	for _, file := range item.Files {
		if file.Path == name {
			return file, true
		}
	}
	return File{}, false
}
