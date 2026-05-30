package worker

import (
	"bakashier/core/message"
	"bakashier/data"
	"errors"
)


func register(msg message.Message, ch chan <- message.Message) error {
	var list data.ManageList
	
	d, ok := msg.Data.(message.InstRegisterData)
	if !ok {
		return errors.New("invalid data type")
	}
	
	err := list.Load(d.ListBks, d.Password)
	if err != nil { return err }
	list.Add(data.ManageListRow{Hash: d.Hash, OrgPath: msg.SrcPath, BksFile: msg.DstPath})
	err = list.Save(d.Password, d.ChunkSize)
	if err != nil { return err }
	
	ch <- message.Message{
		Category: message.MSG_CAT_REPO_REGISTERED,
		SrcPath: msg.SrcPath,
		DstPath: msg.DstPath,
	}
	return nil
}
