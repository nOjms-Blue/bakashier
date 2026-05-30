package message


type InstBackupV1Data struct {
	FileName string
	Password string
	ChunkSize uint64
}

type InstBackupV2Data struct {
	FileName string
	Password string
	ChunkSize uint64
}

type InstRestoreData struct {
	Password string
}

type InstVerifyData struct {
	Password string
	Hash     []byte
}

type InstRegisterData struct {
	ListBks   string
	Hash      []byte
	ChunkSize uint64
	Password  string
}

type RepoHashData struct {
	Hash []byte
}

type RepoVerifiedData struct {
	Verify bool
}
