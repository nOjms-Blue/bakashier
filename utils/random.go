package utils

import "crypto/rand"

const filenameChars = "abcdefghijklmnopqrstuvwxyz0123456789"

// 既存のキーと重複しない英小文字と数字のみのランダムな文字列を生成する。
// existing のキーに同じ文字列が含まれる場合は再生成を繰り返す。
func GenerateUniqueRandomName(existing map[string]string) string {
	const nameLen = 16
	const maxByte = 256 - (256 % len(filenameChars)) // 偏りなく選ぶための上限
	for {
		name := make([]byte, nameLen)
		for i := range name {
			for {
				b := make([]byte, 1)
				if _, err := rand.Read(b); err != nil {
					continue
				}
				if int(b[0]) < maxByte {
					name[i] = filenameChars[int(b[0]) % len(filenameChars)]
					break
				}
			}
		}
		s := string(name)
		if _, exists := existing[s]; !exists {
			return s
		}
	}
}
