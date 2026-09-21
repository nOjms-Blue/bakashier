package archive

import (
	"errors"
	"io"
	"syscall"
	"time"
)

const (
	// maxFileWriteSize は 1 回の Write 上限。
	// Windows の WriteFile は、ネットワークドライブや USB などへ 64MiB 前後
	// （環境によっては 16–32MiB）を一括書き込みすると
	// ERROR_NO_SYSTEM_RESOURCES (1450) を返す。
	maxFileWriteSize = 4 * 1024 * 1024
	minFileWriteSize = 64 * 1024

	maxNoSystemResourcesRetries = 8
)

// windowsErrorNoSystemResources は Win32 の ERROR_NO_SYSTEM_RESOURCES。
const windowsErrorNoSystemResources syscall.Errno = 1450

// テストから待機を無効化できるようにする。
var noSystemResourcesDelay = 10 * time.Millisecond

type limitedWriter struct {
	w        io.Writer
	maxWrite int
}

func newLimitedWriter(w io.Writer) *limitedWriter {
	return newLimitedWriterN(w, maxFileWriteSize)
}

func newLimitedWriterN(w io.Writer, max int) *limitedWriter {
	if max <= 0 {
		max = maxFileWriteSize
	}
	return &limitedWriter{w: w, maxWrite: max}
}

func (w *limitedWriter) Write(p []byte) (int, error) {
	total := 0
	retries := 0
	for len(p) > 0 {
		n := len(p)
		if n > w.maxWrite {
			n = w.maxWrite
		}
		written, err := w.w.Write(p[:n])
		if written > 0 {
			total += written
			p = p[written:]
			retries = 0
		}
		if err == nil {
			if written == 0 {
				return total, io.ErrShortWrite
			}
			continue
		}
		if written == 0 && isWindowsNoSystemResources(err) && retries < maxNoSystemResourcesRetries {
			w.flush()
			if w.maxWrite > minFileWriteSize {
				w.maxWrite = minFileWriteSize
			}
			if noSystemResourcesDelay > 0 {
				time.Sleep(noSystemResourcesDelay)
			}
			retries++
			continue
		}
		return total, err
	}
	return total, nil
}

func (w *limitedWriter) flush() {
	type syncer interface {
		Sync() error
	}
	if s, ok := w.w.(syncer); ok {
		_ = s.Sync()
	}
}

func isWindowsNoSystemResources(err error) bool {
	for err != nil {
		if errno, ok := err.(syscall.Errno); ok {
			return errno == windowsErrorNoSystemResources
		}
		err = errors.Unwrap(err)
	}
	return false
}
