package data

import (
	"bytes"
	"encoding/binary"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const testPassword = "test-password"

// ExportStreamArchive と ImportStreamArchive の往復で内容が一致することを確認する。
func TestStreamArchiveRoundtrip(t *testing.T) {
	tmp := t.TempDir()

	// 複数チャンクにまたがるデータ（チャンクサイズ 1KiB に対して 10KiB + 端数）
	content := bytes.Repeat([]byte("0123456789abcdef"), 640+3)
	srcFile := filepath.Join(tmp, "source.bin")
	if err := os.WriteFile(srcFile, content, 0644); err != nil {
		t.Fatal(err)
	}

	archiveFile := filepath.Join(tmp, "archive.bks")
	if err := ExportStreamArchive(srcFile, archiveFile, "source.bin", testPassword, 1024); err != nil {
		t.Fatalf("ExportStreamArchive failed: %v", err)
	}

	restoreDir := filepath.Join(tmp, "restore")
	if err := os.MkdirAll(restoreDir, 0755); err != nil {
		t.Fatal(err)
	}
	restored, err := ImportStreamArchive(archiveFile, restoreDir, testPassword)
	if err != nil {
		t.Fatalf("ImportStreamArchive failed: %v", err)
	}
	if restored != filepath.Join(restoreDir, "source.bin") {
		t.Fatalf("unexpected restored path: %s", restored)
	}

	restoredContent, err := os.ReadFile(restored)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(content, restoredContent) {
		t.Fatalf("restored content mismatch: want %d bytes, got %d bytes", len(content), len(restoredContent))
	}
}

// 空ファイルがリストアで復元されることを確認する。
func TestStreamArchiveEmptyFile(t *testing.T) {
	tmp := t.TempDir()

	srcFile := filepath.Join(tmp, "empty.txt")
	if err := os.WriteFile(srcFile, []byte{}, 0644); err != nil {
		t.Fatal(err)
	}

	archiveFile := filepath.Join(tmp, "empty.bks")
	if err := ExportStreamArchive(srcFile, archiveFile, "empty.txt", testPassword, 1024); err != nil {
		t.Fatalf("ExportStreamArchive failed: %v", err)
	}

	restoreDir := filepath.Join(tmp, "restore")
	if err := os.MkdirAll(restoreDir, 0755); err != nil {
		t.Fatal(err)
	}
	restored, err := ImportStreamArchive(archiveFile, restoreDir, testPassword)
	if err != nil {
		t.Fatalf("ImportStreamArchive failed: %v", err)
	}

	info, err := os.Stat(restored)
	if err != nil {
		t.Fatalf("restored empty file does not exist: %v", err)
	}
	if info.Size() != 0 {
		t.Fatalf("restored empty file has size %d", info.Size())
	}
}

// 誤ったパスワードでのインポートがエラーになることを確認する。
func TestStreamArchiveWrongPassword(t *testing.T) {
	tmp := t.TempDir()

	srcFile := filepath.Join(tmp, "secret.txt")
	if err := os.WriteFile(srcFile, []byte("secret data"), 0644); err != nil {
		t.Fatal(err)
	}

	archiveFile := filepath.Join(tmp, "secret.bks")
	if err := ExportStreamArchive(srcFile, archiveFile, "secret.txt", testPassword, 1024); err != nil {
		t.Fatal(err)
	}

	restoreDir := filepath.Join(tmp, "restore")
	if err := os.MkdirAll(restoreDir, 0755); err != nil {
		t.Fatal(err)
	}
	if _, err := ImportStreamArchive(archiveFile, restoreDir, "wrong-password"); err == nil {
		t.Fatal("expected error for wrong password, got nil")
	}
}

// 破損・切り詰めされたアーカイブがパニックせずエラーになることを確認する。
func TestImportTruncatedArchive(t *testing.T) {
	tmp := t.TempDir()

	srcFile := filepath.Join(tmp, "source.bin")
	if err := os.WriteFile(srcFile, bytes.Repeat([]byte("x"), 4096), 0644); err != nil {
		t.Fatal(err)
	}
	archiveFile := filepath.Join(tmp, "archive.bks")
	if err := ExportStreamArchive(srcFile, archiveFile, "source.bin", testPassword, 1024); err != nil {
		t.Fatal(err)
	}

	full, err := os.ReadFile(archiveFile)
	if err != nil {
		t.Fatal(err)
	}

	restoreDir := filepath.Join(tmp, "restore")
	if err := os.MkdirAll(restoreDir, 0755); err != nil {
		t.Fatal(err)
	}

	// 空ファイル・ヘッダ途中・名前途中・チャンク途中など、さまざまな位置で切り詰める
	for _, size := range []int{0, 1, 5, 9, 20, len(full) / 2, len(full) - 2} {
		truncated := filepath.Join(tmp, "truncated.bks")
		if err := os.WriteFile(truncated, full[:size], 0644); err != nil {
			t.Fatal(err)
		}
		if _, err := ImportStreamArchive(truncated, restoreDir, testPassword); err == nil {
			t.Fatalf("expected error for archive truncated to %d bytes, got nil", size)
		}
	}
}

// 巨大なチャンク長が宣言されたアーカイブを、メモリ確保前に拒否することを確認する。
func TestImportOversizedChunkLength(t *testing.T) {
	tmp := t.TempDir()

	srcFile := filepath.Join(tmp, "source.bin")
	if err := os.WriteFile(srcFile, []byte("data"), 0644); err != nil {
		t.Fatal(err)
	}
	archiveFile := filepath.Join(tmp, "archive.bks")
	if err := ExportStreamArchive(srcFile, archiveFile, "source.bin", testPassword, 1024); err != nil {
		t.Fatal(err)
	}

	full, err := os.ReadFile(archiveFile)
	if err != nil {
		t.Fatal(err)
	}

	// 名前ブロックの直後にあるチャンク長を巨大な値に書き換える
	nameLen := binary.BigEndian.Uint32(full[5:9])
	chunkLenOffset := 9 + int(nameLen) + 4
	binary.BigEndian.PutUint64(full[chunkLenOffset:chunkLenOffset+8], ^uint64(0))

	tampered := filepath.Join(tmp, "tampered.bks")
	if err := os.WriteFile(tampered, full, 0644); err != nil {
		t.Fatal(err)
	}

	restoreDir := filepath.Join(tmp, "restore")
	if err := os.MkdirAll(restoreDir, 0755); err != nil {
		t.Fatal(err)
	}
	_, err = ImportStreamArchive(tampered, restoreDir, testPassword)
	if err == nil || !strings.Contains(err.Error(), "chunk too large") {
		t.Fatalf("expected 'chunk too large' error, got: %v", err)
	}
}

// アーカイブ内の名前にパス要素が含まれる場合、リストア先の外に書き出されないことを確認する。
func TestImportRejectsUnsafeFileName(t *testing.T) {
	tmp := t.TempDir()

	for _, name := range []string{"../evil.txt", "..", "a/b.txt", `..\evil.txt`, ""} {
		archiveFile := filepath.Join(tmp, "unsafe.bks")
		fp, err := os.Create(archiveFile)
		if err != nil {
			t.Fatal(err)
		}
		content := []byte("malicious")
		offset := 0
		err = ExportBks(
			name,
			func(length uint64) ([]byte, error) {
				if offset >= len(content) {
					return []byte{}, nil
				}
				b := content[offset:]
				offset = len(content)
				return b, nil
			},
			func(data []byte) error {
				_, err := fp.Write(data)
				return err
			},
			testPassword,
			1024,
		)
		fp.Close()
		if err != nil {
			t.Fatal(err)
		}

		restoreDir := filepath.Join(tmp, "restore")
		if err := os.MkdirAll(restoreDir, 0755); err != nil {
			t.Fatal(err)
		}
		if _, err := ImportStreamArchive(archiveFile, restoreDir, testPassword); err == nil {
			t.Fatalf("expected error for unsafe name %q, got nil", name)
		}
		if _, err := os.Stat(filepath.Join(tmp, "evil.txt")); err == nil {
			t.Fatalf("file escaped restore directory for name %q", name)
		}
	}
}

// IsSafeFileName の判定を確認する。
func TestIsSafeFileName(t *testing.T) {
	safe := []string{"file.txt", "ファイル.dat", ".hidden", "a b c"}
	unsafe := []string{"", ".", "..", "a/b", `a\b`, "../x", "/etc/passwd", "a\x00b"}

	for _, name := range safe {
		if !IsSafeFileName(name) {
			t.Errorf("expected %q to be safe", name)
		}
	}
	for _, name := range unsafe {
		if IsSafeFileName(name) {
			t.Errorf("expected %q to be unsafe", name)
		}
	}
}

// ExportBks が writer のエラーを伝播することを確認する。
func TestExportBksPropagatesWriterError(t *testing.T) {
	writeErr := os.ErrClosed
	err := ExportBks(
		"name",
		func(length uint64) ([]byte, error) { return []byte{}, nil },
		func(data []byte) error { return writeErr },
		testPassword,
		1024,
	)
	if err == nil {
		t.Fatal("expected writer error to be propagated, got nil")
	}
}

// ExportBks が不正なチャンクサイズを拒否することを確認する。
func TestExportBksRejectsInvalidChunkSize(t *testing.T) {
	for _, size := range []uint64{0, MAX_CHUNK_SIZE + 1} {
		err := ExportBks(
			"name",
			func(length uint64) ([]byte, error) { return []byte{}, nil },
			func(data []byte) error { return nil },
			testPassword,
			size,
		)
		if err == nil {
			t.Fatalf("expected error for chunk size %d, got nil", size)
		}
	}
}
