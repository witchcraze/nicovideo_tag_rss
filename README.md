# nicovideo_tag_rss

ニコニコ動画の特定のタグ検索結果を定期的に取得・マージし、重複を除去したRSSフィードを配信するHTTPサーバーです。

## 特徴

- **複数タグのマージ**: 1つのRSSフィードに対して複数のタグを指定し、それらの検索結果をマージできます。
- **重複除去**: 複数のタグ検索で重複した動画（動画IDをキーとする）は、1件として処理されます。
- **堅牢なキャッシュ**: タグ情報の取得に失敗した場合でも、前回取得したキャッシュを保持してRSS配信を継続します。
- **キャッシュの永続化**: インメモリキャッシュをディスクに定期的に保存し、サーバー再起動時も前回のキャッシュから復帰します。
- **構造化ログ**: `log/slog` を用いた構造化ログを出力し、運用の監視を容易にします。
- **Docker対応**: Dockerイメージとしてパッケージ化され、GitHub Container Registry等で容易にデプロイ可能です。

## 使い方 (Usage)

### 1. 設定ファイルの作成

`config.yaml` を作成し、配信したいフィードと対象のタグを設定します。

```yaml
listen: ":8080"
update_interval: 15m
cache_dir: "./cache"
video_retention_days: 7
max_pages: 1

feeds:
  - name: "vocaloid"
    title: "VOCALOID Latest"
    description: "Latest VOCALOID videos from Nicovideo"
    tags:
      - "VOCALOID"
      - "初音ミク"

  - name: "game_commentary"
    title: "Game Commentary"
    description: "Latest game commentary videos"
    tags:
      - "ゲーム実況"
```

- `listen`: サーバーがリッスンするアドレス（デフォルト `":8080"`）
- `update_interval`: ニコニコ動画へタグ検索情報を取得しに行く間隔（デフォルト `5m`）
- `cache_dir`: キャッシュファイルを保存するディレクトリ（デフォルト `"./cache"`）。ディレクトリが存在しない場合は自動作成されます。
- `video_retention_days`: キャッシュの feed から古いビデオを削除する期間（日数）。この期間より古い公開日時のビデオは自動削除されます。（デフォルト `7` 日）
- `max_pages`: タグ検索時に取得する最大ページ数。複数ページの検索結果をマージします。（デフォルト `1` ページ、ページングなし）
- `feeds`: 生成するRSSフィードのリスト。`name` がURLパス（例: `/feed/vocaloid`）になります。

### 2. Docker Compose (Nginxリバースプロキシ付き) での起動

GHCRに公開されているイメージと、リバースプロキシとしてのNginxを連携させる `docker-compose.yml` の例です。

```yaml
services:
  app:
    image: ghcr.io/witchcraze/nicovideo_tag_rss:latest
    container_name: nicovideo_rss_app
    restart: always
    volumes:
      # 手元で作成した config.yaml をコンテナの /app/config.yaml にマウント
      - ./config.yaml:/app/config.yaml:ro
      # キャッシュディレクトリをマウントして再起動時も保持
      - ./cache:/app/cache
    # 外部には公開せず、nginx経由でのみアクセスさせる

  nginx:
    image: nginx:alpine
    container_name: nicovideo_rss_nginx
    restart: always
    ports:
      - "80:80"
    volumes:
      - ./nginx.conf:/etc/nginx/conf.d/default.conf:ro
    depends_on:
      - app
```

同じディレクトリに `nginx.conf` を作成します。

```nginx
server {
    listen 80;
    server_name localhost;

    location / {
        proxy_pass http://app:8080;
        
        proxy_set_header Host $host;
        proxy_set_header X-Real-IP $remote_addr;
        proxy_set_header X-Forwarded-For $proxy_add_x_forwarded_for;
        proxy_set_header X-Forwarded-Proto $scheme;
    }
}
```

これらのファイルを配置したディレクトリで以下のコマンドを実行します。

```bash
docker compose up -d
```

起動後、ブラウザで `http://localhost/` にアクセスすると、設定したフィードの一覧が表示されます。
各フィードのRSSは `http://localhost/feed/{name}` で取得できます。

## API 仕様

- `GET /`
  - 登録されているFeedの一覧をHTML形式で返します。
- `GET /feed/{name}`
  - 指定された `name` のRSSフィード（RSS 2.0形式）を返します。ETagによる `304 Not Modified` に対応しています。
  - キャッシュされたデータを返すため高速かつ安定しています。
- `GET /healthz`
  - ヘルスチェック用エンドポイント（200 OK）です。

## 開発 (Development)

ローカルで実行する場合は、Go 1.26以上がインストールされた環境で以下を実行します。

```bash
go run main.go -config config.example.yaml
```

## ライセンス

[MIT License](LICENSE)
