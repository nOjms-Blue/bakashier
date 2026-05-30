package worker

import (
	"bakashier/core/message"
	"bakashier/utils"
)


func calcFileHash(msg message.Message, ch chan<- message.Message) error {
	file := msg.SrcPath
	hash, err := utils.CalcSHA256FromFile(file)
	if err != nil { return err }
	
	ch <- message.Message{
		Category: message.MSG_CAT_REPO_HASH,
		SrcPath: file,
		Data: message.RepoHashData{ Hash: hash },
	}
	return nil
}
