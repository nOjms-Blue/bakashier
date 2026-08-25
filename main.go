// ディレクトリを暗号化・圧縮してバックアップし、復元するCLIツール。
package main

import (
	"errors"
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
		SrcDir:    args.SrcDir,
		DistDir:   args.DistDir,
		Password:  args.Password,
		Workers:   args.Workers,
		ChunkSize: args.ChunkSize,
		Limit:     core.SettingsLimit{Size: args.LimitSize, Wait: args.LimitWait},
	}
	run := func() error {
		if settings.Password == "" {
			input, err := cli.InputPassword()
			if err != nil {
				return err
			}
			settings.Password = input
		}

		wg := sync.WaitGroup{}
		toViewQueue := make(chan view.MessageToView, 64)
		toManagerQueue := make(chan view.MessageToManager, 64)
		viewResult := make(chan error, 1)

		wg.Add(1)
		go func() {
			defer wg.Done()
			model, err := view.Run(args.Mode, toViewQueue, toManagerQueue)
			if err != nil {
				// UI が起動できなくても core の送信を受け続け、処理停止を防ぐ。
				for msg := range toViewQueue {
					if msg.MsgType == view.FINISHED {
						break
					}
				}
				viewResult <- err
				return
			}

			if len(model.ErrorLog) > 0 {
				for _, e := range model.ErrorLog {
					fmt.Println(e)
				}
			}
			viewResult <- nil
		}()
		var operationErr error
		if args.Mode == cli.ModeBackup {
			operationErr = core.Backup(settings, toViewQueue, toManagerQueue)
		} else {
			operationErr = core.Restore(settings, toViewQueue, toManagerQueue)
		}
		wg.Wait()
		return errors.Join(operationErr, <-viewResult)
	}

	switch args.Mode {
	case cli.ModeBackup:
		if err := run(); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	case cli.ModeRestore:
		if err := run(); err != nil {
			fmt.Println(err.Error())
			os.Exit(1)
		}
	case cli.ModeVersion:
		fmt.Println(constants.APP_VERSION)
	case cli.ModeHelp:
		cli.Usage()
	}
}
