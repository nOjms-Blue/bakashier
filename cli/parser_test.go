package cli

import (
	"math"
	"os"
	"path/filepath"
	"strconv"
	"testing"
	"time"
)

func TestIsParentChildDirectoryResolvesSymlinkDestination(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "source")
	nestedDir := filepath.Join(srcDir, "nested")
	distLink := filepath.Join(tmp, "destination-link")
	if err := os.MkdirAll(nestedDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(nestedDir, distLink); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}

	invalid, err := isParentChildDirectory(srcDir, distLink)
	if err != nil {
		t.Fatal(err)
	}
	if !invalid {
		t.Fatal("symlinked destination inside source was not detected")
	}
}

func TestIsParentChildDirectoryResolvesExistingSymlinkAncestor(t *testing.T) {
	tmp := t.TempDir()
	srcDir := filepath.Join(tmp, "source")
	link := filepath.Join(tmp, "link")
	distDir := filepath.Join(link, "not-created-yet")
	if err := os.MkdirAll(srcDir, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(srcDir, link); err != nil {
		t.Skipf("symlink creation is unavailable: %v", err)
	}

	invalid, err := isParentChildDirectory(srcDir, distDir)
	if err != nil {
		t.Fatal(err)
	}
	if !invalid {
		t.Fatal("destination below a symlinked ancestor inside source was not detected")
	}
}

func TestParseArgsRejectsLimitSizeOverflow(t *testing.T) {
	tmp := t.TempDir()
	tooLargeMiB := uint64(math.MaxUint64/(1024*1024) + 1)
	_, err := ParseArgs([]string{
		"--backup",
		filepath.Join(tmp, "source"),
		filepath.Join(tmp, "destination"),
		"--limit-size",
		strconv.FormatUint(tooLargeMiB, 10),
	})
	if err == nil {
		t.Fatal("expected limit size overflow to be rejected")
	}
}

func TestParseArgsRejectsLimitWaitDurationOverflow(t *testing.T) {
	tmp := t.TempDir()
	tooLargeSeconds := uint64(math.MaxInt64/int64(time.Second)) + 1
	_, err := ParseArgs([]string{
		"--backup",
		filepath.Join(tmp, "source"),
		filepath.Join(tmp, "destination"),
		"--limit-wait",
		strconv.FormatUint(tooLargeSeconds, 10),
	})
	if err == nil {
		t.Fatal("expected limit wait duration overflow to be rejected")
	}
}

func TestParseArgsRejectsExcessiveWorkerCount(t *testing.T) {
	tmp := t.TempDir()
	_, err := ParseArgs([]string{
		"--backup",
		filepath.Join(tmp, "source"),
		filepath.Join(tmp, "destination"),
		"--workers",
		"65",
	})
	if err == nil {
		t.Fatal("expected excessive worker count to be rejected")
	}
}

func TestParseArgsRejectsExcessiveConcurrentChunkMemory(t *testing.T) {
	tmp := t.TempDir()
	_, err := ParseArgs([]string{
		"--backup",
		filepath.Join(tmp, "source"),
		filepath.Join(tmp, "destination"),
		"--workers",
		"1",
		"--chunk",
		"8192",
	})
	if err == nil {
		t.Fatal("expected excessive concurrent chunk memory to be rejected")
	}
}

func TestParseArgsAcceptsCommandLinePassword(t *testing.T) {
	tmp := t.TempDir()
	args, err := ParseArgs([]string{
		"--backup",
		filepath.Join(tmp, "source"),
		filepath.Join(tmp, "destination"),
		"--password",
		"secret",
	})
	if err != nil {
		t.Fatal(err)
	}
	if args.Password != "secret" {
		t.Fatalf("password = %q, want %q", args.Password, "secret")
	}
}
