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

func exportArchiveFile(srcFile string, dstFile string, fileName string, password string, chunkSize uint64) error {
	src, err := os.Open(srcFile)
	if err != nil {
		return err
	}
	defer src.Close()

	dst, err := os.Create(dstFile)
	if err != nil {
		return err
	}

	bks := archive.BksArchive{
		Password:  password,
		ChunkSize: chunkSize,
	}
	if err := bks.Export(fileName, src, dst); err != nil {
		dst.Close()
		os.Remove(dstFile)
		return err
	}

	return dst.Close()
}

func importArchiveFile(archiveFile string, dstDirectory string, password string) (string, error) {
	src, err := os.Open(archiveFile)
	if err != nil {
		return "", err
	}
	defer src.Close()

	var dstFile string
	var dst *os.File
	bks := archive.BksArchive{
		ChunkSize: archive.DEFAULT_CHUNK_SIZE,
		Password:  password,
	}
	err = bks.Import(src, func(name string) (io.Writer, error) {
		if !archive.IsSafeFileName(name) {
			return nil, fmt.Errorf("unsafe file name in archive: %q", name)
		}
		dstFile = filepath.Join(dstDirectory, name)
		var createErr error
		dst, createErr = os.Create(dstFile)
		if createErr != nil {
			return nil, createErr
		}
		return dst, nil
	})
	if dst != nil {
		closeErr := dst.Close()
		if err == nil {
			err = closeErr
		}
	}
	if err != nil {
		if dstFile != "" {
			os.Remove(dstFile)
		}
		return "", err
	}
	if dstFile == "" {
		return "", errors.New("archive did not contain any file data")
	}

	return dstFile, nil
}

func saveDirectoryEntries(directoryEntryFile string, name string, entries []archive.DirectoryEntry, password string, chunkSize uint64) error {
	var buf bytes.Buffer
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

	dst, err := os.Create(directoryEntryFile)
	if err != nil {
		return err
	}

	bks := archive.BksArchive{
		Password:  password,
		ChunkSize: chunkSize,
	}
	if err := bks.Export(name, bytes.NewReader(buf.Bytes()), dst); err != nil {
		dst.Close()
		os.Remove(directoryEntryFile)
		return err
	}

	return dst.Close()
}

// _directory_.bks からエントリ一覧を読み込む。ファイルが存在しない場合は空スライスを返す。復号に password を使用する。
func loadDirectoryEntries(directoryEntryFile string, password string) ([]archive.DirectoryEntry, error) {
	if _, err := os.Stat(directoryEntryFile); err != nil {
		return []archive.DirectoryEntry{}, nil
	}

	src, err := os.Open(directoryEntryFile)
	if err != nil {
		return []archive.DirectoryEntry{}, err
	}
	defer src.Close()

	var buf bytes.Buffer
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
	err = archive.ImportDirectoryEntries(bytes.NewReader(buf.Bytes()), func(entry archive.DirectoryEntry) error {
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
