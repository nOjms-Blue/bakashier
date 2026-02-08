# bakashier

[English](README.md) | [日本語](README.ja.md)

`bakashier` is a CLI tool for backing up and restoring directories.  
Backup data is stored in `.bks`, a bakashier-specific custom format, and protected with compression and password-based encryption.

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
- `--help`, `-h`: Show help
- `--version`, `-v`: Show version

### Notes

- `--backup` and `--restore` are mutually exclusive.
- Both `src_dir` and `dist_dir` are required.
- `--password` is required.

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
sh build.sh
```

or:

```bash
go build -o bakashier main.go
```

### Windows

```bat
build.bat
```

or:

```powershell
Start-Process .\build.bat
```

## OSS Libraries Used

- Go Standard Library
- `golang.org/x/crypto` (`v0.47.0`)

See `NOTICE` for license details.
