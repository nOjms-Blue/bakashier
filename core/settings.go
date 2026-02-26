package core


type SettingsLimit struct {
	Size uint64
	Wait uint64
}

type Settings struct {
	SrcDir  string
	DistDir string
	Password string
	Workers uint32
	ChunkSize uint64
	Limit SettingsLimit
}
