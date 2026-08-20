package archive

import (
	"bakashier/utils"
	"encoding/binary"
	"io"
)

// 出力の圧縮・暗号化処理
func exportProcess(chunk []byte, password string) ([]byte, []byte, error) {
	var err error = nil

	// CRC32 ハッシュを計算
	chunkCRC := utils.CRC32HashBytes(chunk)

	// エクスポート用の処理
	chunk, err = utils.CompressBytes(chunk)
	if err != nil {
		return []byte{}, []byte{}, err
	}
	chunk, err = utils.EncryptBytesWithPassword(chunk, password)
	if err != nil {
		return []byte{}, []byte{}, err
	}

	return chunk, chunkCRC, nil
}

func (bks BksArchive) Export(name string, reader io.Reader, writer io.Writer) error {
	chunkSize := bks.ChunkSize
	password := bks.Password

	// ヘッダの bakashier 形式判定用の "BKS"
	_, err := writer.Write([]byte("BKS"))
	if err != nil {
		return err
	}

	// ヘッダの bakashier 形式のファイルバージョン
	var versionBytes = make([]byte, 2)
	binary.BigEndian.PutUint16(versionBytes, 1)
	_, err = writer.Write(versionBytes)
	if err != nil {
		return err
	}

	// ヘッダのファイル名
	var chunkLenBytes = make([]byte, 4)
	chunk, chunkCRC, err := exportProcess([]byte(name), password)
	if err != nil {
		return err
	}
	binary.BigEndian.PutUint32(chunkLenBytes, uint32(len(chunk)))
	if _, err = writer.Write(chunkLenBytes); err != nil {
		return err
	}
	if _, err = writer.Write(chunk); err != nil {
		return err
	}
	if _, err = writer.Write(chunkCRC); err != nil {
		return err
	}

	buf := make([]byte, chunkSize)
	chunkLenBytes = make([]byte, 8)
	for {
		// 1チャンク分読み取り
		n, err := reader.Read(buf)
		if err == io.EOF {
			break
		}
		if err != nil {
			return err
		}

		if n > 0 {
			// 圧縮・暗号化
			chunk, chunkCRC, err = exportProcess(buf[:n], password)
			if err != nil {
				return err
			}

			// 処理後のチャンク長を bytes に変換
			binary.BigEndian.PutUint64(chunkLenBytes, uint64(len(chunk)))

			// 書き込み
			if _, err = writer.Write(chunkLenBytes); err != nil {
				return err
			}
			if _, err := writer.Write(chunk); err != nil {
				return err
			}
			if _, err := writer.Write(chunkCRC); err != nil {
				return err
			}
		}
	}

	return nil
}
