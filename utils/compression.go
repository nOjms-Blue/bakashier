package utils

import (
	"bytes"
	"compress/zlib"
	"errors"
	"io"
	"math"
)

var ErrDecompressedDataTooLarge = errors.New("decompressed data exceeds limit")

// バイト配列を zlib で圧縮し、結果のバイト列を返す。
func CompressBytes(data []byte) ([]byte, error) {
	var b bytes.Buffer
	w := zlib.NewWriter(&b)
	_, err := w.Write(data)
	if err != nil {
		return nil, err
	}
	if err := w.Close(); err != nil {
		return nil, err
	}
	return b.Bytes(), nil
}

// zlib で圧縮されたバイト配列を展開して返す。
func DecompressBytes(compressedData []byte) ([]byte, error) {
	var out bytes.Buffer
	if err := DecompressBytesToWriter(compressedData, &out, math.MaxInt64-1); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// zlib で圧縮されたバイト配列を maxSize バイトまで展開して writer に書き込む。
func DecompressBytesToWriter(compressedData []byte, writer io.Writer, maxSize uint64) error {
	if maxSize >= math.MaxInt64 {
		return errors.New("decompression limit is too large")
	}
	b := bytes.NewReader(compressedData)
	r, err := zlib.NewReader(b)
	if err != nil {
		return err
	}
	limited := io.LimitReader(r, int64(maxSize)+1)
	written, err := io.Copy(writer, limited)
	if err != nil {
		_ = r.Close()
		return err
	}
	if err := r.Close(); err != nil {
		return err
	}
	if uint64(written) > maxSize {
		return ErrDecompressedDataTooLarge
	}
	return nil
}
