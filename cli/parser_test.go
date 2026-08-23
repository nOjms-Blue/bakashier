package cli

import (
	"os"
	"path/filepath"
	"testing"
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
