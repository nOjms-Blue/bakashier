package cli

import (
	"fmt"
	
	"bakashier/constants"
)


// コマンドラインの使い方を標準出力に表示する。
func Usage() {
	fmt.Println("Usage:")
	fmt.Printf("  %s [--backup|-b|--restore|-r] [src_dir] [dist_dir] --password|-p [password]\n", constants.APP_NAME)
	fmt.Printf("  %s [--help|-h|--version|-v]\n", constants.APP_NAME)
	fmt.Println("")
	fmt.Println("  --backup, -b      Run backup")
	fmt.Println("  --restore, -r     Run restore")
	fmt.Println("  --password, -p    Required password")
	fmt.Println("  --chunk, -c       Chunk size in MiB for backup (default: 16)")
	fmt.Println("  --limit-size, -ls Limit size in MiB for backup (default: 0)")
	fmt.Println("  --limit-wait, -lw Limit wait in seconds for backup (default: 0)")
	fmt.Println("  --help, -h        Show help")
	fmt.Println("  --version, -v     Show version")
}