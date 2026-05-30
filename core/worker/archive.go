package worker

import (
	"bakashier/core/message"
	"bakashier/data"
	"errors"
)


func backupV1(msg message.Message, ch chan <- message.Message) error {
	d, ok := msg.Data.(message.InstBackupV1Data)
	if !ok {
		return errors.New("invalid data type")
	}
	return data.ExportStreamArchive(msg.SrcPath, msg.DstPath, d.FileName, d.Password, d.ChunkSize)
}

func backupV2(msg message.Message, ch chan <- message.Message) error {
	d, ok := msg.Data.(message.InstBackupV2Data)
	if !ok {
		return errors.New("invalid data type")
	}
	return data.ExportStreamArchive(msg.SrcPath, msg.DstPath, "", d.Password, d.ChunkSize)
}

func restoreV1(msg message.Message, ch chan <- message.Message) error {
	d, ok := msg.Data.(message.InstRestoreData)
	if !ok {
		return errors.New("invalid data type")
	}
	
	restored, err := data.ImportStreamArchive(msg.SrcPath, msg.DstPath, d.Password)
	if err != nil { return err }
	
	ch <- message.Message{
		Category: message.MSG_CAT_REPO_RESTORE_V1,
		DstPath: restored,
	}
	return nil
}

func restoreV2(msg message.Message, ch chan <- message.Message) error {
	d, ok := msg.Data.(message.InstRestoreData)
	if !ok {
		return errors.New("invalid data type")
	}
	return data.ImportStreamArchiveV2(msg.SrcPath, msg.DstPath, d.Password)
}

func verify(msg message.Message, ch chan <- message.Message) error {
	d, ok := msg.Data.(message.InstVerifyData)
	if !ok {
		return errors.New("invalid data type")
	}
	
	verify, err := data.Verify(msg.SrcPath, d.Password, d.Hash)
	if err != nil { return err }
	
	ch <- message.Message{
		Data: message.RepoVerifiedData{Verify: verify},
	}
	return nil
}
