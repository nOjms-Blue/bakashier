# bakashier

[English](README.md) | [日本語](README.ja.md)

`bakashier` is a CLI tool for backing up and restoring directories.  
Backup data is stored in `.bks`, a bakashier-specific custom format, and protected with compression and password-based encryption.

## Features

- Backup and restore directories with a single CLI
- Incremental behavior for unchanged files during backup
- Password-based encryption and compression for archived data
- Optional transfer throttling with `--limit-size` and `--limit-wait`

## Usage

### Command format

```bash
bakashier [--backup|-b|--restore|-r] [src_dir] [dist_dir] --password|-p [password]
bakashier [--help|-h|--version|-v]
```

### Options

- `--backup`, `-b`: Run backup
- `--restore`, `-r`: Run restore
- `--password`, `-p`: Password (required)
- `--chunk`, `-c`: Chunk size in MiB for backup (default: 16)
- `--limit-size`, `-ls`: Limit size in MiB for backup (default: 0 = disabled)
- `--limit-wait`, `-lw`: Limit wait in seconds for backup (default: 0 = disabled)
- `--help`, `-h`: Show help
- `--version`, `-v`: Show version

### Notes

- `--backup` and `--restore` are mutually exclusive.
- Both `src_dir` and `dist_dir` are required.
- `src_dir` and `dist_dir` cannot be parent-child directories.
- `--password` is required.
- `--chunk`, `--limit-size`, and `--limit-wait` require positive integers.

### Examples

```bash
# Backup
bakashier --backup ./src ./dist --password my-secret

# Restore
bakashier --restore ./dist ./restore --password my-secret

# Show version
bakashier --version
```

## Build

### Requirements

- Go 1.25.5 or later (as specified in `go.mod`)

### Linux / macOS

```bash
sh scripts/build.sh
```

or:

```bash
go build -o bakashier main.go
```

### Windows

```bat
scripts\build.bat
```

or:

```powershell
go build -o bakashier.exe main.go
```

## Install

### Linux / macOS

```bash
sh scripts/install.sh
```

Binary install path:

- `${XDG_DATA_HOME:-$HOME/.local/share}/bakashier/bakashier`

### Windows

```bat
scripts\install.bat
```

Binary install path:

- `%LOCALAPPDATA%\bakashier\bakashier.exe`

## OSS Libraries Used

- Go Standard Library
- `github.com/charmbracelet/bubbletea` (`v1.3.10`)
- `github.com/charmbracelet/bubbles` (`v1.0.0`)
- `github.com/charmbracelet/lipgloss` (`v1.1.0`)
- `golang.org/x/crypto` (`v0.47.0`)

See `NOTICE` for license details.
