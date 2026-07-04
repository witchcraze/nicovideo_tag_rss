# nicovideo_tag_rss

ニコニコ動画の特定のタグ検索結果を定期的に取得・マージし、重複を除去したRSSフィードを配信するHTTPサーバーです。

## 特徴

- **複数タグのマージ**: 1つのRSSフィードに対して複数のタグを指定し、それらの検索結果をマージできます。
- **重複除去**: 複数のタグ検索で重複した動画（動画IDをキーとする）は、1件として処理されます。
- **堅牢なキャッシュ**: タグ情報の取得に失敗した場合でも、前回取得したキャッシュを保持してRSS配信を継続します。
- **構造化ログ**: `log/slog` を用いた構造化ログを出力し、運用の監視を容易にします。
- **Docker対応**: Dockerイメージとしてパッケージ化され、GitHub Container Registry等で容易にデプロイ可能です。

## 設定仕様

設定は YAML ファイルで行います。

```yaml
listen: ":8080"
update_interval: 5m

feeds:
  - name: vocaloid
    title: VOCALOID動画
    description: VOCALOID関連動画
    tags:
      - 初音ミク

  - name: music
    title: 音楽まとめ
    description: 歌・演奏動画まとめ
    tags:
      - 初音ミク
      - 歌ってみた
      - 演奏してみた
```

## API 仕様

- `GET /`
  - 登録されているFeedの一覧をHTMLまたはJSONで返します。
- `GET /feed/{name}`
  - 指定された `name` のRSSフィード（RSS 2.0形式）を返します。
  - キャッシュされたデータを返すため高速かつ安定しています。
- `GET /healthz`
  - ヘルスチェック用エンドポイントです。

## ライセンス

[MIT License](LICENSE)
