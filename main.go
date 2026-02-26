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
	
	settings := core.Settings{
		SrcDir: args.SrcDir,
		DistDir: args.DistDir,
		Password: args.Password,
		Workers: args.Workers,
		ChunkSize: args.ChunkSize,
		Limit: core.SettingsLimit{Size: args.LimitSize, Wait: args.LimitWait},
	}
	run := func() {
		if settings.Password == "" {
			input, err := cli.InputPassword()
			if err != nil {
				fmt.Println(err.Error())
				os.Exit(1)
			}
			settings.Password = input
		}
		
		wg := sync.WaitGroup{}
		toViewQueue := make(chan view.MessageToView, 64)
		toManagerQueue := make(chan view.MessageToManager, 64)
		
		wg.Add(1)
		go func() {
			defer wg.Done()
			model, err := view.Run(args.Mode, toViewQueue, toManagerQueue)
			if err != nil {
				fmt.Println(err.Error())
				return
			}
			
			if len(model.ErrorLog) > 0 {
				for _, e := range model.ErrorLog {
					fmt.Println(e)
				}
			}
		}()
		if args.Mode == cli.ModeBackup {
			core.Backup(settings, toViewQueue, toManagerQueue)
		} else {
			core.Restore(settings, toViewQueue, toManagerQueue)
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
