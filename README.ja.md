# bakashier

[English](README.md) | [日本語](README.ja.md)

`bakashier` は、ディレクトリをバックアップ/リストアするための CLI ツールです。  
バックアップデータは bakashier 独自形式の `.bks` で保存され、圧縮とパスワード暗号化が行われます。

## 使い方

### コマンド形式

```bash
bakashier [--backup|-b|--restore|-r] [src_dir] [dist_dir] --password|-p [password]
bakashier [--help|-h|--version|-v]
```

### オプション

- `--backup`, `-b`: バックアップを実行
- `--restore`, `-r`: リストアを実行
- `--password`, `-p`: パスワード（必須）
- `--help`, `-h`: ヘルプ表示
- `--version`, `-v`: バージョン表示

### 注意事項

- `--backup` と `--restore` は同時指定できません。
- `src_dir` と `dist_dir` は必須です。
- `--password` は必須です（省略不可）。

### 実行例

```bash
# バックアップ
bakashier --backup ./src ./dist --password my-secret

# リストア
bakashier --restore ./dist ./restore --password my-secret

# バージョン表示
bakashier --version
```

## ビルド方法

### 前提

- Go 1.25.5 以降（`go.mod` の指定に準拠）

### Linux / macOS

```bash
sh build.sh
```

または:

```bash
go build -o bakashier main.go
```

### Windows

```bat
build.bat
```

または:

```powershell
go build -o bakashier.exe main.go
```

## 使用している OSS ライブラリ

- Go Standard Library
- `golang.org/x/crypto` (`v0.47.0`)

ライセンス情報は `NOTICE` を参照してください。
