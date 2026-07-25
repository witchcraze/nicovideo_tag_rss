# Prometheus と Grafana を用いた監視ダッシュボード

本アプリケーション (`nicovideo_tag_rss`) は、Prometheus 形式のメトリクスを `/metrics` エンドポイントで提供しています。
これにより、Prometheusでデータを収集し、Grafanaで視覚化することができます。

## 1. Prometheus の設定

Prometheus でメトリクスを収集するための設定例です。
`prometheus.yml` に以下のジョブを追加してください。

```yaml
scrape_configs:
  - job_name: 'nicovideo_tag_rss'
    scrape_interval: 15s
    static_configs:
      - targets: ['localhost:8080'] # アプリケーションのアドレスに合わせて変更してください
```

## 2. Grafana でのダッシュボード作成

Grafana で簡単に状態を監視できるダッシュボードのサンプル (`dashboard.json`) を用意しています。

### インポート手順

1. Grafana にログインし、サイドメニューから **Dashboards** > **New** > **Import** を選択します。
2. 「Upload JSON file」をクリックし、本リポジトリ内の `docs/dashboard.json` をアップロードします。
   （または、ファイルの内容をコピーしてテキストエリアに貼り付け、「Load」をクリックします）
3. データソースとして、接続済みの Prometheus を選択して「Import」をクリックします。

### ダッシュボードで確認できる主な指標

- **HTTP Requests**: エンドポイント別・ステータスコード別のリクエストレート
- **Nico Requests & Retries**: ニコニコ動画へのリクエスト数とリトライの発生回数
- **Cache Hit Rate**: キャッシュのヒット・ミス・NotModified (304) の割合
- **HTML Parse**: HTMLパースの成功・失敗レート
- **Feed Update Duration**: バックグラウンドでのRSSフィード更新にかかる時間のヒストグラム
