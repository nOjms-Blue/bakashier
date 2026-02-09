package cli

import (
	"fmt"
	"strconv"
	
	"bakashier/data"
)


// コマンドライン引数を解析し、モード・ソースディレクトリ・出力先・パスワード・チャンクサイズを返す。
// エラー時は第6戻り値にエラーを返し、help/version の場合は特別なエラー文字列を使用する。
func ParseArgs(args []string) (ParsedArgs, error) {
	var mode ModeType
	var srcDir string
	var distDir string
	var password string
	var chunkSizeMiB uint64 = uint64(0) // 0 = 未指定（デフォルト使用）
	var limitSizeMiB uint64 = uint64(0) // 0 = 未指定（デフォルト使用）
	var limitWaitSec uint64 = uint64(0) // 0 = 未指定（デフォルト使用）
	positional := make([]string, 0, 2)
	
	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--backup", "-b":
			if mode != "" && mode != ModeBackup {
				return ParsedArgs{}, fmt.Errorf("cannot use backup and restore at the same time")
			}
			mode = ModeBackup
		case "--restore", "-r":
			if mode != "" && mode != ModeRestore {
				return ParsedArgs{}, fmt.Errorf("cannot use backup and restore at the same time")
			}
			mode = ModeRestore
		case "--password", "-p":
			if i+1 >= len(args) {
				return ParsedArgs{}, fmt.Errorf("password value is required")
			}
			next := args[i+1]
			if len(next) == 0 || next[0] == '-' {
				return ParsedArgs{}, fmt.Errorf("password value is required")
			}
			password = next
			i++
		case "--chunk", "-c":
			if i+1 >= len(args) {
				return ParsedArgs{}, fmt.Errorf("chunk size value is required")
			}
			chunkArg := args[i+1]
			if len(chunkArg) == 0 || chunkArg[0] == '-' {
				return ParsedArgs{}, fmt.Errorf("chunk size value is required")
			}
			parsed, err := strconv.ParseUint(chunkArg, 10, 64)
			if err != nil || parsed == 0 {
				return ParsedArgs{}, fmt.Errorf("chunk size must be a positive integer (MiB)")
			}
			chunkSizeMiB = parsed
			i++
		case "--limit-size", "-ls":
			if i+1 >= len(args) {
				return ParsedArgs{}, fmt.Errorf("limit size value is required")
			}
			limitSizeArg := args[i+1]
			if len(limitSizeArg) == 0 || limitSizeArg[0] == '-' {
				return ParsedArgs{}, fmt.Errorf("limit size value is required")
			}
			parsed, err := strconv.ParseUint(limitSizeArg, 10, 64)
			if err != nil || parsed == 0 {
				return ParsedArgs{}, fmt.Errorf("limit size must be a positive integer (MiB)")
			}
			limitSizeMiB = parsed * 1024 * 1024
			i++
		case "--limit-wait", "-lw":
			if i+1 >= len(args) {
				return ParsedArgs{}, fmt.Errorf("limit wait value is required")
			}
			limitWaitArg := args[i+1]
			if len(limitWaitArg) == 0 || limitWaitArg[0] == '-' {
				return ParsedArgs{}, fmt.Errorf("limit wait value is required")
			}
			parsed, err := strconv.ParseUint(limitWaitArg, 10, 64)
			if err != nil || parsed == 0 {
				return ParsedArgs{}, fmt.Errorf("limit wait must be a positive integer (seconds)")
			}
			limitWaitSec = parsed
			i++
		case "--help", "-h":
			return ParsedArgs{Mode: ModeHelp}, nil
		case "--version", "-v":
			return ParsedArgs{Mode: ModeVersion}, nil
		default:
			if len(arg) > 0 && arg[0] == '-' {
				return ParsedArgs{}, fmt.Errorf("unknown option: %s", arg)
			}
			positional = append(positional, arg)
		}
	}

	if mode == "" {
		return ParsedArgs{}, fmt.Errorf("backup or restore mode is required")
	}
	if password == "" {
		return ParsedArgs{}, fmt.Errorf("password is required")
	}
	if len(positional) < 2 {
		return ParsedArgs{}, fmt.Errorf("src_dir and dist_dir are required")
	}
	if len(positional) > 2 {
		return ParsedArgs{}, fmt.Errorf("too many positional arguments")
	}

	srcDir = positional[0]
	distDir = positional[1]

	chunkSize := data.ChunkSize
	if chunkSizeMiB > 0 {
		chunkSize = chunkSizeMiB * 1024 * 1024
	} else {
		chunkSize = data.ChunkSize
	}

	return ParsedArgs{
		Mode:      mode,
		SrcDir:    srcDir,
		DistDir:   distDir,
		Password:  password,
		ChunkSize: chunkSize,
		LimitSize: limitSizeMiB,
		LimitWait: limitWaitSec,
	}, nil
}