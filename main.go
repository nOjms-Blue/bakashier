// ディレクトリを暗号化・圧縮してバックアップし、復元するCLIツール。
package main

import (
	"fmt"
	"os"
	
	"bakashier/constants"
	"bakashier/core"
	"bakashier/cli"
)


func main() {
	args, err := cli.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Println(err.Error())
		cli.Usage()
		os.Exit(1)
	}
	
	limit := core.Limit{Size: args.LimitSize, Wait: args.LimitWait}
	
	switch args.Mode {
	case cli.ModeBackup:
		core.Backup(args.SrcDir, args.DistDir, args.Password, args.ChunkSize, limit)
	case cli.ModeRestore:
		core.Restore(args.SrcDir, args.DistDir, args.Password, limit)
	case cli.ModeVersion:
		fmt.Println(constants.APP_VERSION)
	case cli.ModeHelp:
		cli.Usage()
	}
}
