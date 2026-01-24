# Reazy

Reazy (Read + Lazy) は、Go と Bubble Tea で構築されたモダンなターミナルベースの RSS リーダーです。シンプルな Vim ライクなインターフェースを提供し、お気に入りの RSS フィードをコマンドラインから直接管理・閲覧できます。

## 特徴

- **TUI インターフェース**: シンプルでレスポンシブなターミナル UI。
- **フィード管理**: RSS フィードの追加と削除が簡単に行えます。
- **閲覧**: フィードアイテムを閲覧し、デフォルトブラウザで記事を開くことができます。
- **Vim キーバインド**: `j`, `k`, `h`, `l` でのナビゲーション。
- **カスタマイズ可能**: YAML でキーバインドやフィードリストを設定可能。
- **更新機能**: プルリフレッシュスタイルの更新をサポート。
- **既読管理**: 読んだ記事を追跡し、薄く表示します。
- **全フィード表示**: 全てのフィードの記事を一つのタイムラインで表示します。
- **記事プレビュー**: ブラウザを開く前に、ターミナル内で記事の要約を確認できます。

## インストール

### ソースコードから
```bash
go install github.com/tesso57/reazy/cmd/reazy@latest
```

または、クローンしてビルド:
```bash
git clone https://github.com/tesso57/reazy.git
cd reazy
go build ./cmd/reazy
```

## 使い方

アプリケーションを実行します:
```bash
reazy
```

### キーバインド (デフォルト)
- **ナビゲーション**:
  - `k` / `↑`: 上へ移動
  - `j` / `↓`: 下へ移動
  - `h` / `←`: 戻る / フィード一覧へフォーカス
  - `l` / `→` / `Enter`: サマリーを表示 / リンクを開く
- **アクション**:
  - `a`: フィードを追加
  - `x`: フィードを削除
  - `r`: フィードを更新
  - `?`: ヘルプの切り替え
  - `q`: 終了

## 設定
設定ファイルは `$XDG_CONFIG_HOME/reazy/config.yaml` (通常は `~/.config/reazy/config.yaml`) に保存されます。

例:
```yaml
feeds:
  - https://news.ycombinator.com/rss
keymap:
  up: k
  down: j
  ...
```


## 類似のプロジェクト
他にもRSSリーダーが存在します:
- [eilmeldung](https://github.com/christo-auer/eilmeldung)
- [russ](https://github.com/ckampfe/russ)

## 開発
このプロジェクトではタスク管理に `xc` を使用しています。

- 実行: `xc run`
- テスト: `xc test`
- カバレッジ: `xc cover`
- クリーン: `xc clean`
