# Architecture

このドキュメントは「上に新しい設計を追記する」方式で更新します。
最新の設計が上に来るように、バージョン単位で記録します。

## v1 (Current: DDD)
現状の構成を踏まえ、DDD的な境界とレイヤリングで整理した構成です。

### 層の考え方
- Domain: ビジネスルールの中心。外部依存を持たない。
- Application: ユースケースの調整役。Domainを組み合わせて実行する。
- Infrastructure: 外部I/Oや永続化の実装。Domain/Applicationのインターフェースに従う。
- Presentation: UI/CLI/TUIなど入力と表示の層。

### 各レイヤの構成と責務
#### Presentation
Presentation層はユーザー入力を解釈し、画面状態を更新し、Application層から受け取ったデータを表示用に整形して描画に渡す。
ここでUIのインタラクション全体を完結させる。
- `internal/presentation/tui/model.go`: 画面状態と入力処理の中心。画面遷移やCmd発行を行う。
- `internal/presentation/tui/container.go`: `model` から描画用のPropsを組み立てる。
- `internal/presentation/tui/state/`: UI状態のみを保持する（画面種別、選択状態、モーダル表示、入力中など）。
- `internal/presentation/tui/intent/`: 入力(KeyMsg)を意図(Intent)に変換する。
- `internal/presentation/tui/update/`: Intent + State から新しい State と Command を導出する。
- `internal/presentation/tui/presenter/`: 表示用データの整形（list.Item生成、並び替え、ラベル付与）。
- `internal/presentation/tui/components/`: 見た目の部品（header/main/sidebar/modal など）。
- `internal/presentation/tui/view/`: 画面全体のレイアウト/描画ロジック。
- `internal/presentation/tui/view/list/`: list.Item の描画委譲（feed/article の見た目）。

#### Application
Application層はユースケースの流れを組み立て、Domainを使って処理の手順を表現する。
- `internal/application/usecase/`: UIが呼び出すユースケース（購読操作・取得・履歴反映・AI要約/タグ生成）と、AI要約/タグ向けのプロンプト生成・応答パースを扱う。
- `internal/application/settings/`: 設定値の型（keymap/theme/feeds など）。

#### Domain
Domain層はビジネスルールと中核モデルを保持し、外部依存を持たない。
- `internal/domain/reading/`: 記事・フィード・履歴など読み取りドメインの中核モデル。
- `internal/domain/subscription/`: 購読モデル（feed URL など）。

#### Infrastructure
Infrastructure層は外部I/Oや永続化の実装を提供し、Application/Domainから参照される。
- `internal/infrastructure/feed/`: RSS取得・パース（gofeed）。
- `internal/infrastructure/history/`: 履歴の永続化（JSONL）。
- `internal/infrastructure/config/`: 設定の読み書き（kong + yaml）。
- `internal/infrastructure/ai/`: AIプロバイダ連携の抽象化と実装（例: Codex CLI）。

### ディレクトリ構造
```
cmd/
  reazy/
    main.go

internal/
  domain/
    reading/
      feed.go
      history.go
    subscription/
      subscription.go

  application/
    settings/
      settings.go
    usecase/
      reading.go
      subscription.go
      insight.go
      insight_generator.go

  infrastructure/
    feed/
      feed.go
    config/
      config.go
    history/
      history.go
    ai/
      codexcli/
        client.go

  presentation/
    tui/
      model.go
      container.go
      state/
      intent/
      update/
      presenter/
      components/
      view/
        list/
```

### 依存方向
```
presentation -> application -> domain
infrastructure -----------^
```

### データフロー
フロント(presentation)とバックエンド(application/domain/infrastructure)の往復を明確にします。

```
User Input
  ↓
Intent Parser
  ↓
Reducer (State update)
  ↓                 ↘
Presenter/ViewModel  Command (backend)
  ↓                   ↓
View/Render          Backend Result (Msg)
  ↓                   ↓
UIに表示             Reducer (State update)
                      ↓
                    Presenter/ViewModel
                      ↓
                    View/Render
```

代表例:
- フィード更新
  - Intent: Refresh
  - State: loading=true
  - Command: FetchFeed
  - Msg: FeedFetched
  - State更新 → 表示用データ再構築 → Render
- フィード追加
  - Intent: StartAddFeed / ConfirmAddFeed
  - Command: AddSubscription + SaveConfig
  - Msg: SubscriptionUpdated
  - State更新 → フィード一覧再構築 → Render
- 記事を開く
  - Intent: OpenArticle
  - State: detailView
  - Command: MarkRead + SaveHistory
  - State更新 → 記事一覧再構築 → Render
- AI要約/タグ生成
  - Intent: Summarize
  - State: loading=true
  - Command: GenerateInsight(AI client abstraction -> Codex subprocess)
  - Msg: InsightGenerated
  - State更新（History + 記事表示）→ SaveHistory → Render

### Migration Guide (v0 -> v1)
移行を段階化して、動作を維持しながら責務を分離していくためのガイドです。

1. ドメインの抽出
   - `internal/feed`, `internal/history` から純粋なモデルを切り出し `internal/domain` に配置する。
   - 外部依存(gofeed/JSONL)に触れる処理は残さない。

2. アプリケーション層の導入
   - UIから呼ばれている処理をユースケース単位で `internal/application/usecase` に移す。
   - UIは「入力の解釈」と「出力の表示」に絞る。

3. インフラ層の切り出し
   - RSS取得は `internal/infrastructure/feed` に移動。
   - 履歴永続化は `internal/infrastructure/history` に移動。
   - 設定I/Oは `internal/infrastructure/config` に移動。

4. 依存方向の固定
   - `domain` は他層に依存しない。
   - `application` は `domain` にのみ依存。
   - `presentation` と `infrastructure` は `application` / `domain` に依存。

5. 接続点の整理
   - リポジトリ/ゲートウェイのインターフェースを `domain` or `application` に定義。
   - 実装は `infrastructure` に置く。

---

## v0 (Previous)

v0 時点のReazyのアーキテクチャ(設計)を整理したものです。
DDD移行前の層と責務、依存関係を記録しています。

### 全体像
Reazyは1つのバイナリとして動作するTUIアプリです。
エントリポイントで設定を読み込み、TUI層が入力・描画・データ取得をまとめて調整します。

### 層の説明(現状)
#### 1. Entry / Composition
- 場所: `cmd/reazy/main.go`
- 役割: 設定ロード後にTUIモデルを生成し、Bubble Teaの実行を開始する。
- 依存: `internal/config`, `internal/ui`

#### 2. Presentation (TUI)
- 場所: `internal/ui`
- 役割: 画面状態、キーバインド、描画、ユーザー操作のハンドリング。
- 備考: 現状はユースケースのオーケストレーションもここに含まれる。
- 依存: `internal/config`, `internal/feed`, `internal/history` ほかBubble Tea関連

#### 3. Feed Ingestion
- 場所: `internal/feed`
- 役割: RSSの取得とパース、複数Feedの集約、並列取得。
- 依存: `gofeed`

#### 4. History Persistence
- 場所: `internal/history`
- 役割: 既読などの履歴をJSONLで読み書き。
- 依存: `encoding/json`, `os`

#### 5. Configuration
- 場所: `internal/config`
- 役割: YAML設定のロード/保存、デフォルト値の補完。
- 依存: `kong`, `yaml.v3`

### 依存方向(現状)
UIが中心にあり、外部連携や永続化を呼び出す構成です。循環依存はありません。

```
cmd/reazy
  -> internal/ui
       -> internal/config
       -> internal/feed
       -> internal/history
```

### データフロー(例)
- 起動時
  1) `cmd/reazy` が `config.LoadConfig()` を実行
  2) `ui.NewModel(cfg)` でUIを構築し、TUIを起動

- フィード更新
  1) UIが `feed.Fetch` / `feed.FetchAll` を呼び出す
  2) 取得したアイテムを履歴(`history.Manager`)と突き合わせ
  3) 表示用リストを更新

- フィード追加/削除
  1) UIが `config.AddFeed` / `config.RemoveFeed` を呼び出す
  2) `config.Save` により設定ファイルへ永続化

### 主要ディレクトリ
```
cmd/reazy
internal/config
internal/feed
internal/history
internal/ui
```

### 補足
現状はTUI層がユースケースをまとめて扱うため、アプリケーション層とUI層が近い構成です。
今後モジュラーモノリスやDDD構成へ移行する場合は、ユースケースを切り出すと責務分離が進みます。
