package archive


type BksArchive struct {
	Password string
	ChunkSize int64
}

// エクスポート時に指定できるプレーンテキストチャンクの上限サイズ。
const MAX_CHUNK_SIZE uint64 = 8 * 1024 * 1024 * 1024 // 8GiB

// インポート時に許容する圧縮・暗号化済みチャンクの上限サイズ（zlib・AES-GCM のオーバーヘッドを考慮）。
const maxEncryptedChunkSize uint64 = MAX_CHUNK_SIZE + MAX_CHUNK_SIZE/128 + 1024

// インポート時に許容する圧縮・暗号化済み名前ブロックの上限サイズ。
const maxEncryptedNameSize uint64 = 64 * 1024
