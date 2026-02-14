package utils

import (
	"bytes"
	"compress/zlib"
	"io"
)


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
	b := bytes.NewReader(compressedData)
	r, err := zlib.NewReader(b)
	if err != nil {
		return nil, err
	}
	defer r.Close()
	var out bytes.Buffer
	_, err = io.Copy(&out, r)
	if err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}