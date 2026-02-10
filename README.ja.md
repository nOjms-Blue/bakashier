# bakashier

[English](README.md) | [日本語](README.ja.md)

`bakashier` は、ディレクトリをバックアップ/リストアするための CLI ツールです。  
バックアップデータは bakashier 独自形式の `.bks` で保存され、圧縮とパスワード暗号化が行われます。

## 主な機能

- ディレクトリのバックアップ/リストアを 1 つの CLI で実行
- バックアップ時に変更のないファイルをスキップ
- パスワード暗号化と圧縮によるアーカイブ保護
- `--limit-size` と `--limit-wait` による処理制限

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
- `--chunk`, `-c`: バックアップ時のチャンクサイズ（MiB、デフォルト: 16）
- `--limit-size`, `-ls`: バックアップ時のサイズ制限（MiB、デフォルト: 0 = 無効）
- `--limit-wait`, `-lw`: バックアップ時の待機時間制限（秒、デフォルト: 0 = 無効）
- `--help`, `-h`: ヘルプ表示
- `--version`, `-v`: バージョン表示

### 注意事項

- `--backup` と `--restore` は同時指定できません。
- `src_dir` と `dist_dir` は必須です。
- `src_dir` と `dist_dir` は親子ディレクトリ関係にできません。
- `--password` は必須です（省略不可）。
- `--chunk`、`--limit-size`、`--limit-wait` は正の整数を指定してください。

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
sh scripts/build.sh
```

または:

```bash
go build -o bakashier main.go
```

### Windows

```bat
scripts\build.bat
```

または:

```powershell
go build -o bakashier.exe main.go
```

## インストール

### Linux / macOS

```bash
sh scripts/install.sh
```

バイナリの配置先:

- `${XDG_DATA_HOME:-$HOME/.local/share}/bakashier/bakashier`

### Windows

```bat
scripts\install.bat
```

バイナリの配置先:

- `%LOCALAPPDATA%\bakashier\bakashier.exe`

## 使用している OSS ライブラリ

- Go Standard Library
- `golang.org/x/crypto` (`v0.47.0`)

ライセンス情報は `NOTICE` を参照してください。
