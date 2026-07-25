## 関連Issue
Fixes #63

## 変更内容
`feed.GenerateRSS` を直接呼び出していた箇所を、`RSSGenerator` インターフェースを介すようにリファクタリングしました。
- `feed/rss.go`: `RSSGenerator` インターフェースと `DefaultRSSGenerator` 実装を追加。
- `feed/aggregator.go`: `NewAggregator` でジェネレーターを受け取れるように変更。省略時はデフォルト実装を使用。
- `feed/aggregator_test.go`: `mockRSSGenerator` を使って、RSSの生成に失敗した場合でも古いキャッシュが保持され、エラーを返すことをテスト (`TestAggregator_Update_RSSGenerationError`) 。
- 各種テストや `main.go` の修正。

## セルフレビュー用チェックリスト
- [x] TDDとテスト網羅性: 既存のテストに加え、エラー時の挙動確認テストを追加しパスさせました
- [x] コード品質とエラーハンドリング: DIによってテストコードの独立性を高めました
- [x] 設計方針への準拠: インターフェースベースの設計に揃えています
