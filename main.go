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
		wg := sync.WaitGroup{}
		toViewQueue := make(chan view.MessageToView, 64)
		toDispatcherQueue := make(chan view.MessageToDispatcher, 64)
		program := view.NewProgram(args.Mode, toViewQueue, toDispatcherQueue)
		
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
			core.Backup(args.SrcDir, args.DistDir, args.Password, args.ChunkSize, limit, toViewQueue, toDispatcherQueue)
		} else {
			core.Restore(args.SrcDir, args.DistDir, args.Password, limit, toViewQueue, toDispatcherQueue)
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
