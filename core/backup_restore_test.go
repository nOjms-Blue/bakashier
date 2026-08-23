package core

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"bakashier/view"
)

// Backup と Restore を実行し、ビューへのメッセージを破棄しながら完了を待つ。
func runWithDrainedView(t *testing.T, run func(toView chan<- view.MessageToView, fromView <-chan view.MessageToManager)) []string {
	t.Helper()

	toView := make(chan view.MessageToView, 64)
	fromView := make(chan view.MessageToManager, 64)

	errorLog := make(chan string, 1024)
	drainDone := make(chan struct{})
	go func() {
		defer close(drainDone)
		for msg := range toView {
			if msg.MsgType == view.ERROR {
				errorLog <- msg.Detail
			}
		}
	}()

	run(toView, fromView)
	close(toView)
	<-drainDone
	close(errorLog)

	errors := []string{}
	for e := range errorLog {
		errors = append(errors, e)
	}
	return errors
}

// バックアップ→リストアの往復で、ディレクトリ構造と内容が一致することを確認する。
func TestBackupRestoreRoundtrip(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	backupDir := filepath.Join(tmp, "backup")
	restoreDir := filepath.Join(tmp, "restore")

	// テスト用のディレクトリツリーを作成（空ファイル・空ディレクトリ・ネストを含む）
	files := map[string][]byte{
		"a.txt":            []byte("hello world"),
		"empty.txt":        {},
		"sub/b.bin":        bytes.Repeat([]byte{0xde, 0xad, 0xbe, 0xef}, 1024),
		"sub/deep/c.txt":   []byte("nested file"),
		"sub2/日本語ファイル.txt": []byte("日本語の内容"),
	}
	if err := os.MkdirAll(filepath.Join(srcDir, "emptydir"), 0755); err != nil {
		t.Fatal(err)
	}
	for name, content := range files {
		path := filepath.Join(srcDir, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, content, 0644); err != nil {
			t.Fatal(err)
		}
	}

	settings := Settings{
		SrcDir:    srcDir,
		DistDir:   backupDir,
		Password:  "test-password",
		Workers:   2,
		ChunkSize: 1024,
	}

	// バックアップ
	errors := runWithDrainedView(t, func(toView chan<- view.MessageToView, fromView <-chan view.MessageToManager) {
		Backup(settings, toView, fromView)
	})
	if len(errors) > 0 {
		t.Fatalf("backup reported errors: %v", errors)
	}

	// リストア
	settings.SrcDir = backupDir
	settings.DistDir = restoreDir
	errors = runWithDrainedView(t, func(toView chan<- view.MessageToView, fromView <-chan view.MessageToManager) {
		Restore(settings, toView, fromView)
	})
	if len(errors) > 0 {
		t.Fatalf("restore reported errors: %v", errors)
	}

	// 内容の検証
	for name, content := range files {
		path := filepath.Join(restoreDir, filepath.FromSlash(name))
		restored, err := os.ReadFile(path)
		if err != nil {
			t.Errorf("restored file %s missing: %v", name, err)
			continue
		}
		if !bytes.Equal(content, restored) {
			t.Errorf("restored file %s content mismatch: want %d bytes, got %d bytes", name, len(content), len(restored))
		}
	}
	if _, err := os.Stat(filepath.Join(restoreDir, "emptydir")); err != nil {
		t.Errorf("restored empty directory missing: %v", err)
	}
}

func TestBackupRejectsNonEmptyUninitializedDestination(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	backupDir := filepath.Join(tmp, "backup")
	unrelatedFile := filepath.Join(backupDir, "unrelated.txt")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(srcDir, "source.txt"), []byte("source"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unrelatedFile, []byte("must remain"), 0644); err != nil {
		t.Fatal(err)
	}

	settings := Settings{
		SrcDir:    srcDir,
		DistDir:   backupDir,
		Password:  "test-password",
		Workers:   1,
		ChunkSize: 1024,
	}
	errors := runWithDrainedView(t, func(toView chan<- view.MessageToView, fromView <-chan view.MessageToManager) {
		Backup(settings, toView, fromView)
	})

	if len(errors) == 0 {
		t.Fatal("expected backup to reject a non-empty uninitialized destination")
	}
	got, err := os.ReadFile(unrelatedFile)
	if err != nil {
		t.Fatalf("unrelated destination file was removed: %v", err)
	}
	if string(got) != "must remain" {
		t.Fatalf("unrelated destination file changed: got %q", got)
	}
}

func TestBackupPreservesDestinationWhenMetadataCannotBeInspected(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	backupDir := filepath.Join(tmp, "backup")
	metadataFile := filepath.Join(backupDir, "_directory_.bks")
	unrelatedFile := filepath.Join(backupDir, "unrelated.txt")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(unrelatedFile, []byte("must remain"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("_directory_.bks", metadataFile); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}

	settings := Settings{
		SrcDir:    srcDir,
		DistDir:   backupDir,
		Password:  "test-password",
		Workers:   1,
		ChunkSize: 1024,
	}
	errors := runWithDrainedView(t, func(toView chan<- view.MessageToView, fromView <-chan view.MessageToManager) {
		Backup(settings, toView, fromView)
	})

	if len(errors) == 0 {
		t.Fatal("expected backup to report an unreadable metadata file")
	}
	if _, err := os.Lstat(metadataFile); err != nil {
		t.Fatalf("metadata symlink was removed: %v", err)
	}
	got, err := os.ReadFile(unrelatedFile)
	if err != nil {
		t.Fatalf("unrelated destination file was removed: %v", err)
	}
	if string(got) != "must remain" {
		t.Fatalf("unrelated destination file changed: got %q", got)
	}
}

func TestFailedIncrementalBackupPreservesPreviousFileVersion(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "src")
	backupDir := filepath.Join(tmp, "backup")
	restoreDir := filepath.Join(tmp, "restore")
	srcFile := filepath.Join(srcDir, "source.txt")

	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(srcFile, []byte("previous version"), 0644); err != nil {
		t.Fatal(err)
	}
	settings := Settings{
		SrcDir:    srcDir,
		DistDir:   backupDir,
		Password:  "test-password",
		Workers:   1,
		ChunkSize: 1024,
	}
	errors := runWithDrainedView(t, func(toView chan<- view.MessageToView, fromView <-chan view.MessageToManager) {
		Backup(settings, toView, fromView)
	})
	if len(errors) != 0 {
		t.Fatalf("initial backup reported errors: %v", errors)
	}

	if err := os.WriteFile(srcFile, []byte("new version"), 0644); err != nil {
		t.Fatal(err)
	}
	settings.ChunkSize = 0
	errors = runWithDrainedView(t, func(toView chan<- view.MessageToView, fromView <-chan view.MessageToManager) {
		Backup(settings, toView, fromView)
	})
	if len(errors) == 0 {
		t.Fatal("expected incremental backup failure for invalid chunk size")
	}

	restoreSettings := Settings{
		SrcDir:    backupDir,
		DistDir:   restoreDir,
		Password:  "test-password",
		Workers:   1,
		ChunkSize: 1024,
	}
	errors = runWithDrainedView(t, func(toView chan<- view.MessageToView, fromView <-chan view.MessageToManager) {
		Restore(restoreSettings, toView, fromView)
	})
	if len(errors) != 0 {
		t.Fatalf("restore of previous backup reported errors: %v", errors)
	}
	got, err := os.ReadFile(filepath.Join(restoreDir, "source.txt"))
	if err != nil {
		t.Fatalf("previous file version could not be restored: %v", err)
	}
	if string(got) != "previous version" {
		t.Fatalf("restored content = %q, want previous version", got)
	}
}

func TestBackupReturnsWorkerErrors(t *testing.T) {
	tmp := t.TempDir()
	settings := Settings{
		SrcDir:    filepath.Join(tmp, "missing-source"),
		DistDir:   filepath.Join(tmp, "backup"),
		Password:  "test-password",
		Workers:   1,
		ChunkSize: 1024,
	}

	var operationErr error
	messages := runWithDrainedView(t, func(toView chan<- view.MessageToView, fromView <-chan view.MessageToManager) {
		operationErr = Backup(settings, toView, fromView)
	})
	if operationErr == nil {
		t.Fatalf("Backup returned nil despite worker errors: %v", messages)
	}
}

func TestRestoreReturnsWorkerErrors(t *testing.T) {
	tmp := t.TempDir()
	settings := Settings{
		SrcDir:    filepath.Join(tmp, "missing-backup"),
		DistDir:   filepath.Join(tmp, "restore"),
		Password:  "test-password",
		Workers:   1,
		ChunkSize: 1024,
	}

	var operationErr error
	messages := runWithDrainedView(t, func(toView chan<- view.MessageToView, fromView <-chan view.MessageToManager) {
		operationErr = Restore(settings, toView, fromView)
	})
	if operationErr == nil {
		t.Fatalf("Restore returned nil despite worker errors: %v", messages)
	}
}
