package main

import (
	"fmt"
	"os"

	"bakashier/constants"
	"bakashier/core"
)

type modeType string

const (
	modeBackup  modeType = "backup"
	modeRestore modeType = "restore"
	modeVersion modeType = "version"
)

func usage() {
	fmt.Println("Usage:")
	fmt.Printf("  %s [--backup|-b|--restore|-r] [src_dir] [dist_dir] --password|-p [password]\n", constants.APP_NAME)
	fmt.Printf("  %s [--help|-h|--version|-v]\n", constants.APP_NAME)
	fmt.Println("")
	fmt.Println("  --backup, -b   Run backup")
	fmt.Println("  --restore, -r  Run restore")
	fmt.Println("  --password, -p Required password")
	fmt.Println("  --help, -h     Show help")
	fmt.Println("  --version, -v  Show version")
}

func parseArgs(args []string) (modeType, string, string, string, error) {
	var mode modeType
	var srcDir string
	var distDir string
	password := ""
	positional := make([]string, 0, 2)

	for i := 0; i < len(args); i++ {
		arg := args[i]
		switch arg {
		case "--backup", "-b":
			if mode != "" && mode != modeBackup {
				return "", "", "", "", fmt.Errorf("cannot use backup and restore at the same time")
			}
			mode = modeBackup
		case "--restore", "-r":
			if mode != "" && mode != modeRestore {
				return "", "", "", "", fmt.Errorf("cannot use backup and restore at the same time")
			}
			mode = modeRestore
		case "--password", "-p":
			if i+1 >= len(args) {
				return "", "", "", "", fmt.Errorf("password value is required")
			}
			next := args[i+1]
			if len(next) == 0 || next[0] == '-' {
				return "", "", "", "", fmt.Errorf("password value is required")
			}
			password = next
			i++
		case "--help", "-h":
			return "", "", "", "", fmt.Errorf("help")
		case "--version", "-v":
			return modeVersion, "", "", "", nil
		default:
			if len(arg) > 0 && arg[0] == '-' {
				return "", "", "", "", fmt.Errorf("unknown option: %s", arg)
			}
			positional = append(positional, arg)
		}
	}

	if mode == "" {
		return "", "", "", "", fmt.Errorf("backup or restore mode is required")
	}
	if password == "" {
		return "", "", "", "", fmt.Errorf("password is required")
	}
	if len(positional) < 2 {
		return "", "", "", "", fmt.Errorf("src_dir and dist_dir are required")
	}
	if len(positional) > 2 {
		return "", "", "", "", fmt.Errorf("too many positional arguments")
	}

	srcDir = positional[0]
	distDir = positional[1]

	return mode, srcDir, distDir, password, nil
}

func main() {
	mode, srcDir, distDir, password, err := parseArgs(os.Args[1:])
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
		core.Backup(srcDir, distDir, password)
	case modeRestore:
		core.Restore(srcDir, distDir, password)
	case modeVersion:
		fmt.Println(constants.APP_VERSION)
	}
}
