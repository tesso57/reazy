# Reazy

Reazy (Read + Lazy) は、Go と Bubble Tea で構築されたモダンなターミナルベースの RSS/Atom リーダーです。シンプルな Vim ライクなインターフェースを提供し、お気に入りのフィードをコマンドラインから直接管理・閲覧できます。

## 特徴

- **TUI インターフェース**: シンプルでレスポンシブなターミナル UI。
- **フィード管理**: RSS/Atom フィードの追加と削除が簡単に行えます。
- **閲覧**: フィードアイテムを閲覧し、デフォルトブラウザで記事を開くことができます。
- **Vim キーバインド**: `j`, `k`, `h`, `l` でのナビゲーション。
- **カスタマイズ可能**: YAML でキーバインドやフィードリストを設定可能。
- **更新機能**: プルリフレッシュスタイルの更新をサポート。
- **既読管理**: 読んだ記事を追跡し、薄く表示します。
- **全フィード表示**: 全てのフィードの記事を一つのタイムラインで表示します。
- **AI要約ビュー**: 詳細画面で AI 要約と本文を明確に分けて表示し、読みやすくします。
- **AI インサイト（任意）**: Codex CLI を使って記事の要約とタグを生成できます。
- **AI ステータスフッター**: 記事一覧/詳細画面では AI 生成ステータスをフッターに表示します。

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
  - `l` / `→` / `Enter`: AI 要約 + 本文を表示 / リンクを開く
- **アクション**:
  - `a`: フィードを追加
  - `x`: フィードを削除
  - `r`: フィードを更新
  - `b`: ブックマーク切り替え
  - `s`: AI 要約/タグを生成
  - `S`: AI要約の表示/非表示を切り替え（詳細画面）
  - `?`: ヘルプの切り替え
  - `q`: 終了

## 設定
設定ファイルは `$XDG_CONFIG_HOME/reazy/config.yaml` (通常は `~/.config/reazy/config.yaml`) に保存されます。

例:
```yaml
feeds:
  - https://news.ycombinator.com/rss
  - https://github.com/golang/go/releases.atom
keymap:
  up: k
  down: j
  ...
codex:
  enabled: false
  command: codex
  model: gpt-5
  web_search: disabled
  reasoning_effort: low
  reasoning_summary: none
  verbosity: low
  timeout_seconds: 30
  sandbox: read-only
```

### Codex 連携（任意）
Codex CLI がインストール済み・ログイン済みなら、次の設定で有効化できます。

```yaml
codex:
  enabled: true
```

記事一覧/詳細画面で `s` キーを押すと、以下を生成します。
- 3分程度で読める日本語要約
- 英語のトピックタグ

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
