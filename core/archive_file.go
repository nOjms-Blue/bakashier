package core

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"bakashier/archive"
)

const maxDirectoryEntriesBytes = 64 * 1024 * 1024
const maxDirectoryEntryCount = 1_000_000

var errDirectoryEntriesTooLarge = errors.New("directory entries exceed size limit")

type limitedBuffer struct {
	buffer  bytes.Buffer
	maxSize int
}

func (w *limitedBuffer) Write(p []byte) (int, error) {
	if len(p) > w.maxSize-w.buffer.Len() {
		return 0, errDirectoryEntriesTooLarge
	}
	return w.buffer.Write(p)
}

func writeFileAtomically(path string, write func(io.Writer) error) error {
	tmp, err := os.CreateTemp(filepath.Dir(path), ".bakashier-*")
	if err != nil {
		return err
	}
	tmpPath := tmp.Name()
	defer func() {
		_ = tmp.Close()
		_ = os.Remove(tmpPath)
	}()

	if err := write(tmp); err != nil {
		return err
	}
	if err := tmp.Sync(); err != nil {
		return err
	}
	if err := tmp.Close(); err != nil {
		return err
	}
	if err := os.Rename(tmpPath, path); err != nil {
		return err
	}
	return nil
}

func exportArchiveFile(srcFile string, dstFile string, fileName string, password string, chunkSize uint64) error {
	src, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer src.Close()

	bks := archive.BksArchive{
		Password:  password,
		ChunkSize: chunkSize,
	}
	return writeFileAtomically(dstFile, func(dst io.Writer) error {
		return bks.Export(fileName, src, dst)
	})
}

func importArchiveFile(archiveFile string, dstDirectory string, password string) (string, error) {
	return importArchiveFileWithExpectedName(archiveFile, dstDirectory, password, "")
}

func importArchiveFileWithExpectedName(archiveFile string, dstDirectory string, password string, expectedName string) (string, error) {
	src, err := os.Open(archiveFile)
	if err != nil {
		return "", err
	}
	defer src.Close()

	var dstFile string
	var dst *os.File
	var tmpFile string
	bks := archive.BksArchive{
		ChunkSize: archive.DEFAULT_CHUNK_SIZE,
		Password:  password,
	}
	err = bks.Import(src, func(name string) (io.Writer, error) {
		if !archive.IsSafeFileName(name) {
			return nil, fmt.Errorf("unsafe file name in archive: %q", name)
		}
		if expectedName != "" && name != expectedName {
			return nil, fmt.Errorf("archive file name mismatch: got %q, want %q", name, expectedName)
		}
		dstFile = filepath.Join(dstDirectory, name)
		var createErr error
		dst, createErr = os.CreateTemp(dstDirectory, ".bakashier-restore-*")
		if createErr != nil {
			return nil, createErr
		}
		tmpFile = dst.Name()
		return dst, nil
	})
	if dst != nil {
		syncErr := dst.Sync()
		closeErr := dst.Close()
		if err == nil {
			err = syncErr
		}
		if err == nil {
			err = closeErr
		}
	}
	if err != nil {
		if tmpFile != "" {
			_ = os.Remove(tmpFile)
		}
		return "", err
	}
	if dstFile == "" {
		return "", errors.New("archive did not contain any file data")
	}
	if err := os.Rename(tmpFile, dstFile); err != nil {
		_ = os.Remove(tmpFile)
		return "", err
	}

	return dstFile, nil
}

func saveDirectoryEntries(directoryEntryFile string, name string, entries []archive.DirectoryEntry, password string, chunkSize uint64) error {
	if len(entries) > maxDirectoryEntryCount {
		return errDirectoryEntriesTooLarge
	}
	buf := limitedBuffer{maxSize: maxDirectoryEntriesBytes}
	index := 0
	err := archive.ExportDirectoryEntries(func(entry *archive.DirectoryEntry) error {
		if index >= len(entries) {
			return io.EOF
		}
		*entry = entries[index]
		index++
		return nil
	}, &buf)
	if err != nil {
		return err
	}

	bks := archive.BksArchive{
		Password:  password,
		ChunkSize: chunkSize,
	}
	return writeFileAtomically(directoryEntryFile, func(dst io.Writer) error {
		return bks.Export(name, bytes.NewReader(buf.buffer.Bytes()), dst)
	})
}

// _directory_.bks からエントリ一覧を読み込む。ファイルが存在しない場合は空スライスを返す。復号に password を使用する。
func loadDirectoryEntries(directoryEntryFile string, password string) ([]archive.DirectoryEntry, error) {
	src, err := os.Open(directoryEntryFile)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return []archive.DirectoryEntry{}, nil
		}
		return []archive.DirectoryEntry{}, err
	}
	defer src.Close()

	buf := limitedBuffer{maxSize: maxDirectoryEntriesBytes}
	bks := archive.BksArchive{
		ChunkSize: archive.DEFAULT_CHUNK_SIZE,
		Password:  password,
	}
	err = bks.Import(src, func(name string) (io.Writer, error) {
		return &buf, nil
	})
	if err != nil {
		return []archive.DirectoryEntry{}, err
	}

	var entries []archive.DirectoryEntry
	err = archive.ImportDirectoryEntries(bytes.NewReader(buf.buffer.Bytes()), func(entry archive.DirectoryEntry) error {
		if len(entries) >= maxDirectoryEntryCount {
			return errDirectoryEntriesTooLarge
		}
		if entry.Type != archive.File && entry.Type != archive.Directory {
			return fmt.Errorf("invalid directory entry type: %q", entry.Type)
		}
		if !archive.IsSafeFileName(entry.RealName) || !archive.IsSafeFileName(entry.HideName) {
			return fmt.Errorf("unsafe directory entry name: real=%q hide=%q", entry.RealName, entry.HideName)
		}
		entries = append(entries, entry)
		return nil
	})
	if err != nil {
		return []archive.DirectoryEntry{}, err
	}
	if entries == nil {
		entries = []archive.DirectoryEntry{}
	}
	return entries, nil
}
