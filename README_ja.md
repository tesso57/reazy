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
- **通常一覧の日付セクション**: `All Feeds` / `Bookmarks` / 各フィード一覧を日付ごとに分けて表示します。
- **Newsタブ（AIダイジェスト）**: 登録フィードの「当日記事」から AI が日次ニューストピックを生成し、日付ごとの履歴として保持します。News更新時は同日分の過去トピックを残したまま新規追加します。
- **SQLite履歴保存**: 既読状態・ブックマーク・AI情報をSQLiteへ保存し、起動時/更新時の体感を改善します。
- **AI要約ビュー**: 詳細画面で AI 要約と本文を明確に分けて表示し、読みやすくします。
- **AI インサイト（任意）**: Codex CLI を使って記事の要約とタグを生成できます。
- **ステータスフッター**: AI 生成ステータスに加え、フィードのタイムアウト件数などを表示します。

## インストール

ソースコードからビルドする場合は Go 1.26 以上が必要です。

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

フィードサイドバーの `* News` を選ぶと、日付ごとに保持された AI ニューストピック履歴を表示できます。  
当日分は登録済みフィードから生成され、同日中はキャッシュ利用されます。  
`News` で手動更新すると、当日ダイジェストを再生成しつつ同日分の過去トピックも保持します。  
通常のフィード一覧（`All Feeds` / `Bookmarks` / 各フィード）は日付セクションで表示されます。
一部フィードが遅い場合は、取得できた結果を先に表示し、タイムアウト件数をフッターに表示します。

### キーバインド (デフォルト)
- **ナビゲーション**:
  - `k` / `↑`: 上へ移動
  - `j` / `↓`: 下へ移動
  - `h` / `←`: 戻る / フィード一覧へフォーカス
  - `l` / `→` / `Enter`: 選択中アイテムを開く（記事/ニューストピック/詳細内リンク）
- **アクション**:
  - `a`: フィードを追加
  - `x`: フィードを削除
  - `r`: 現在のフィードを更新（`News` では当日ダイジェストを再生成し、同日分の過去トピックを保持）
  - `b`: ブックマーク切り替え
  - `s`: AI 要約/タグを生成（記事一覧/詳細）
  - `S`: AI要約の表示/非表示を切り替え（詳細画面）
  - `?`: ヘルプの切り替え
  - `q`: 終了

## 設定
設定ファイルは `$XDG_CONFIG_HOME/reazy/config.yaml` (通常は `~/.config/reazy/config.yaml`) に保存されます。
`history_file` のデフォルトは `~/.local/share/reazy/history.db` です。
`history_file` に `.jsonl` を指定している場合でも、同じディレクトリの `history.db` が利用されます。

例:
```yaml
feeds:
  - https://news.ycombinator.com/rss
  - https://github.com/golang/go/releases.atom
keymap:
  up: k
  down: j
  ...
history_file: /Users/you/.local/share/reazy/history.db
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

`* News` を開くと、過去日付分を含む AI ニューストピック履歴を確認できます。

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
