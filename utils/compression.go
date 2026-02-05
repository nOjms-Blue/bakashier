package utils

import (
	"bytes"
	"compress/zlib"
	"io"
)

// byte配列をzlibで圧縮する
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

// zlibで圧縮されたbyte配列を展開する
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