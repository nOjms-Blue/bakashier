package worker

import "bakashier/core/message"


func Worker(password string, chunkSize uint64, submitCh chan<- message.Message, receiveCh <-chan message.Message) {
	var err error = nil
	var isFinish bool = false
	
	for {
		msg := <-receiveCh
		
		switch msg.Category {
		case message.MSG_CAT_INST_SEARCH:
			err = searchTask(msg, submitCh)
		case message.MSG_CAT_INST_BACKUP_V1:
			err = backupV1(msg, submitCh)
		case message.MSG_CAT_INST_BACKUP_V2:
			err = backupV2(msg, submitCh)
		case message.MSG_CAT_INST_RESTORE_V1:
			err = restoreV1(msg, submitCh)
		case message.MSG_CAT_INST_RESTORE_V2:
			err = restoreV2(msg, submitCh)
		case message.MSG_CAT_INST_FILE_HASH:
			err = calcFileHash(msg, submitCh)
		case message.MSG_CAT_INST_VERIFY:
			err = verify(msg, submitCh)
		case message.MSG_CAT_INST_REGISTER:
			err = register(msg, submitCh)
		case message.MSG_CAT_INST_FINISH:
			isFinish = true
		}
		
		if isFinish { break }
		if err != nil {
			submitCh <- message.Message{
				Category: message.MSG_CAT_REPO_ERROR,
				Data: err,
			}
		}
	}
}
