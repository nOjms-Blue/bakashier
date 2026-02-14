package utils

import "crypto/rand"


const filenameChars = "abcdefghijklmnopqrstuvwxyz0123456789"

// existing に存在しない、英小文字と数字のみのランダムな文字列を返す。
// 重複する場合は再生成を繰り返す。暗号論的乱数を使用する。
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
