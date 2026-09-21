package core

import (
	"fmt"
	"os"
)

func rejectSymlinkPath(path string) error {
	info, err := os.Lstat(path)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return err
	}
	if info.Mode()&os.ModeSymlink != 0 {
		return fmt.Errorf("symbolic link is not allowed: %q", path)
	}
	return nil
}
