// ディレクトリを暗号化・圧縮してバックアップし、復元するCLIツール。
package main

import (
	"fmt"
	"os"
	"sync"
	
	"bakashier/cli"
	"bakashier/constants"
	"bakashier/core"
	"bakashier/view"
)


func main() {
	args, err := cli.ParseArgs(os.Args[1:])
	if err != nil {
		fmt.Println(err.Error())
		cli.Usage()
		os.Exit(1)
	}
	
	limit := core.Limit{Size: args.LimitSize, Wait: args.LimitWait}
	run := func() {
		if args.Password == "" {
			input, err := cli.InputPassword()
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			args.Password = input
		}
		
		wg := sync.WaitGroup{}
		toViewQueue := make(chan view.MessageToView, 64)
		toManagerQueue := make(chan view.MessageToManager, 64)
		program := view.NewProgram(args.Mode, toViewQueue, toManagerQueue)
		
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := program.Run()
			if err != nil {
				fmt.Println(err.Error())
				return
			}
		}()
		if args.Mode == cli.ModeBackup {
			core.Backup(args.SrcDir, args.DistDir, args.Password, args.ChunkSize, limit, toViewQueue, toManagerQueue)
		} else {
			core.Restore(args.SrcDir, args.DistDir, args.Password, limit, toViewQueue, toManagerQueue)
		}
		wg.Wait()
	}
	
	switch args.Mode {
	case cli.ModeBackup:
		run()
	case cli.ModeRestore:
		run()
	case cli.ModeVersion:
		fmt.Println(constants.APP_VERSION)
	case cli.ModeHelp:
		cli.Usage()
	}
}
