// ディレクトリを暗号化・圧縮してバックアップし、復元するCLIツール。
package main

import (
	"fmt"
	"os"
	"strconv"

	"bakashier/constants"
	"bakashier/core"
	"bakashier/data"
)

// アプリケーションの動作モード（バックアップ/復元/バージョン表示）。
type modeType string

const (
	modeBackup  modeType = "backup"
	modeRestore modeType = "restore"
	modeVersion modeType = "version"
)

// コマンドラインの使い方を標準出力に表示する。
func usage() {
	fmt.Println("Usage:")
	fmt.Printf("  %s [--backup|-b|--restore|-r] [src_dir] [dist_dir] --password|-p [password]\n", constants.APP_NAME)
	fmt.Printf("  %s [--help|-h|--version|-v]\n", constants.APP_NAME)
	fmt.Println("")
	fmt.Println("  --backup, -b   Run backup")
	fmt.Println("  --restore, -r  Run restore")
	fmt.Println("  --password, -p Required password")
	fmt.Println("  --chunk, -c    Chunk size in MiB for backup (default: 16)")
	fmt.Println("  --help, -h     Show help")
	fmt.Println("  --version, -v  Show version")
}

// コマンドライン引数を解析し、モード・ソースディレクトリ・出力先・パスワード・チャンクサイズを返す。
// エラー時は第6戻り値にエラーを返し、help/version の場合は特別なエラー文字列を使用する。
func parseArgs(args []string) (modeType, string, string, string, uint64, error) {
	var mode modeType
	var srcDir string
	var distDir string
	password := ""
	chunkSizeMiB := uint64(0) // 0 = 未指定（デフォルト使用）
	positional := make([]string, 0, 2)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--backup", "-b":
			if mode != "" && mode != modeBackup {
				return "", "", "", "", 0, fmt.Errorf("cannot use backup and restore at the same time")
			}
			mode = modeBackup
		case "--restore", "-r":
			if mode != "" && mode != modeRestore {
				return "", "", "", "", 0, fmt.Errorf("cannot use backup and restore at the same time")
			}
			mode = modeRestore
		case "--password", "-p":
			if i+1 >= len(args) {
				return "", "", "", "", 0, fmt.Errorf("password value is required")
			}
			next := args[i+1]
			if len(next) == 0 || next[0] == '-' {
				return "", "", "", "", 0, fmt.Errorf("password value is required")
			}
			password = next
			i++
		case "--chunk", "-c":
			if i+1 >= len(args) {
				return "", "", "", "", 0, fmt.Errorf("chunk size value is required")
			}
			chunkArg := args[i+1]
			if len(chunkArg) == 0 || chunkArg[0] == '-' {
				return "", "", "", "", 0, fmt.Errorf("chunk size value is required")
			}
			parsed, err := strconv.ParseUint(chunkArg, 10, 64)
			if err != nil || parsed == 0 {
				return "", "", "", "", 0, fmt.Errorf("chunk size must be a positive integer (MiB)")
			}
			chunkSizeMiB = parsed
			i++
		case "--help", "-h":
			return "", "", "", "", 0, fmt.Errorf("help")
		case "--version", "-v":
			return modeVersion, "", "", "", 0, nil
		default:
			if len(arg) > 0 && arg[0] == '-' {
				return "", "", "", "", 0, fmt.Errorf("unknown option: %s", arg)
			}
			positional = append(positional, arg)
		}
	}

	if mode == "" {
		return "", "", "", "", 0, fmt.Errorf("backup or restore mode is required")
	}
	if password == "" {
		return "", "", "", "", 0, fmt.Errorf("password is required")
	}
	if len(positional) < 2 {
		return "", "", "", "", 0, fmt.Errorf("src_dir and dist_dir are required")
	}
	if len(positional) > 2 {
		return "", "", "", "", 0, fmt.Errorf("too many positional arguments")
	}

	srcDir = positional[0]
	distDir = positional[1]

	chunkSize := data.ChunkSize
	if chunkSizeMiB > 0 {
		chunkSize = chunkSizeMiB * 1024 * 1024
	} else {
		chunkSize = data.ChunkSize
	}

	return mode, srcDir, distDir, password, chunkSize, nil
}

// エントリポイント。引数解析後にバックアップまたは復元を実行する。
func main() {
	mode, srcDir, distDir, password, chunkSize, err := parseArgs(os.Args[1:])
	if err != nil {
		if err.Error() == "help" {
			usage()
			return
		}
		fmt.Println(err.Error())
		usage()
		os.Exit(1)
	}

	switch mode {
	case modeBackup:
		core.Backup(srcDir, distDir, password, chunkSize)
	case modeRestore:
		core.Restore(srcDir, distDir, password)
	case modeVersion:
		fmt.Println(constants.APP_VERSION)
	}
}
