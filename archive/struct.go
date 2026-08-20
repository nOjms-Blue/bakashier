package archive

import (
	"path/filepath"
	"strings"
)

type BksArchive struct {
	Password  string
	ChunkSize int64
}

// デフォルトのチャンクサイズ
const DEFAULT_CHUNK_SIZE uint64 = 16 * 1024 * 1024 // 16MiB

// エクスポート時に指定できるプレーンテキストチャンクの上限サイズ。
const MAX_CHUNK_SIZE uint64 = 8 * 1024 * 1024 * 1024 // 8GiB

// インポート時に許容する圧縮・暗号化済みチャンクの上限サイズ（zlib・AES-GCM のオーバーヘッドを考慮）。
const maxEncryptedChunkSize uint64 = MAX_CHUNK_SIZE + MAX_CHUNK_SIZE/128 + 1024

// インポート時に許容する圧縮・暗号化済み名前ブロックの上限サイズ。
const maxEncryptedNameSize uint64 = 64 * 1024

// アーカイブ由来の名前が、ディレクトリ要素を含まない安全な単一のファイル名かを検証する。
// パストラバーサル（zip-slip）対策として、リストア時の書き出し先の組み立て前に使用する。
func IsSafeFileName(name string) bool {
	if name == "" || name == "." || name == ".." {
		return false
	}
	if strings.ContainsAny(name, "/\\") || strings.ContainsRune(name, 0) {
		return false
	}
	if name != filepath.Base(name) {
		return false
	}
	return true
}
