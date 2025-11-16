# chat-for-websocket-study

[![CI](https://github.com/t-chov/chat-for-websocket-study/actions/workflows/ci.yml/badge.svg)](https://github.com/t-chov/chat-for-websocket-study/actions/workflows/ci.yml)

WebSocket を用いたチャットルームを Go で実装した学習用リポジトリです。`cmd/server` でチャットルーム（WebSocket サーバー）を、`cmd/client` で CLI クライアントを提供し、双方が `internal/chat` にある共通ロジック（トークン生成、チェックサム検証、メッセージのファンアウトなど）を共有します。

## 特徴

- チャット ID（6 桁 + チェックディジット）とソルトを用いて発行する MD5 トークンでメッセージ送信者を検証
- WebSocket 接続上の JSON プロトコルを mermaid シーケンス図付きでドキュメント化（`docs/sequence.md`）
- `/healthz` によるヘルスチェックと INFO ログで観測しやすい WebSocket サーバー
- CLI クライアントを複数起動してローカルでブロードキャストの挙動を確認可能

## 必要要件

- Go 1.25 以降
- Git とターミナル環境（CLI からの操作を想定）

初回は依存モジュールを取得してください。

```bash
go mod download
```

## ディレクトリ構成

```
cmd/
  server/   WebSocket ルームを提供する HTTP サーバー
  client/   CLI 参加者。サーバーから発行されたトークンを使って送受信
internal/chat/
  *.go      ルーム管理、WebSocket ハンドラー、トークン生成などの共通ロジック
docs/
  sequence.md  プロトコルとシーケンス図
```

## 使い方

### サーバー

```bash
go run ./cmd/server \
  --port 28080 \
  --chat-id 1234564 \
  --salt oAQF6zsVq7xg3sd6 \
  --ws-path /ws
```

- `--port`: 受信ポート（既定: 28080）
- `--chat-id`: 参加を許可するチャット ID。クライアントは同じ ID を使う必要があります
- `--salt`: トークン生成に用いるソルト。実運用では環境変数などで秘匿してください
- `--ws-path`: WebSocket エンドポイントのパス

サーバーは `/healthz` で疎通確認ができます。

### クライアント

クライアントは起動時に表示名（`--name`）が必須です。複数ターミナルで名前を変えて起動すると、同じ部屋で会話を再現できます。

```bash
go run ./cmd/client \
  --server ws://localhost:28080/ws \
  --chat-id 1234564 \
  --name alice
```

- 最初に `join` メッセージを送信して部屋へ参加し、サーバーからトークンを受領した後にチャットを開始します
- 入力した行はそのまま `message` として送られ、他クライアントへブロードキャストされます

### シーケンスとプロトコル

- 参加～トークン発行～メッセージ送信までの詳細なやり取りは `docs/sequence.md` を参照してください
- プロトコルは `internal/chat/message.go` の `Envelope` 構造体に従った JSON を用います

## 開発・テスト

品質チェックの基本コマンド:

```bash
go test ./... -cover
golangci-lint run
```

挙動確認はサーバーを 1 つ起動し、クライアントを最低 2 つ起動してブロードキャストとトークン検証を確かめてください。

## ライセンス

このリポジトリは [MIT License](LICENSE) の下で公開されています。
