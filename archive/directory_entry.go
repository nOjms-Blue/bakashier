package archive

import (
	"time"
)

// エントリがディレクトリかファイルかを表す。
type DirectoryEntryType byte

const (
	Unknown   DirectoryEntryType = 'U'
	Directory DirectoryEntryType = 'D'
	File      DirectoryEntryType = 'F'
)

// 1つのファイルまたはディレクトリの実名・隠し名・サイズ・更新日時を保持する。
type DirectoryEntry struct {
	Type     DirectoryEntryType
	RealName string
	HideName string
	Size     uint64
	ModTime  time.Time
}

// ファイル名の最大長
const MAX_NAME_SIZE = 1024
