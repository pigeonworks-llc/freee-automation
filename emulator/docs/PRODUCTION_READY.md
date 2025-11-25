# 本番環境への移行ガイド

## ❌ 元のコードの問題点

### 1. **非推奨APIの使用**
```go
io.ReadAll(resp.Body)      // ❌ ioutil.ReadAll は deprecated
oauth2.NoContext           // ❌ deprecated、context.Background() を使用
```

### 2. **エラーハンドリング不足**
- API障害時のリトライがない
- タイムアウト設定がない
- エラー時のロギングが不十分

### 3. **OAuth2実装の問題**
```go
token := &oauth2.Token{AccessToken: os.Getenv("FREE_ACCESS_TOKEN")}
```
- トークンのリフレッシュ処理がない
- トークンの永続化がない
- エミュレータと本番で認証方法が異なる

### 4. **運用面の問題**
- 構造化ログがない
- メトリクス収集がない
- 異常検知の閾値がない
- デプロイ・監視の仕組みがない

## ✅ 改善版の特徴

### 1. **エラーハンドリング強化**
```go
// リトライロジック
maxRetries := 3
for i := 0; i < maxRetries; i++ {
    resp, err = client.Do(req)
    if err == nil && resp.StatusCode < 500 {
        break
    }
    time.Sleep(time.Second * time.Duration(i+1))
}
```

### 2. **タイムアウト設定**
```go
ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
defer cancel()
```

### 3. **環境別実行**
```go
// エミュレータモード
if cfg.ClientID == "" {
    // シンプルなBearer Token認証
}

// 本番モード
else {
    // OAuth2認証
}
```

### 4. **詳細ログ出力**
```go
for _, txn := range txns {
    log.Printf("  [未仕訳] ID:%d, 金額:%d, 摘要:%s",
        txn.ID, txn.Amount, txn.Description)
}
```

### 5. **異常検知**
```go
if unclassifiedCount > 10 {
    log.Printf("Warning: Too many unclassified transactions")
    os.Exit(1) // 監視システムへアラート
}
```

## 🚀 本番環境デプロイ

### 必要な環境変数

#### 本番環境
```bash
# freee OAuth2設定
export FREEE_CLIENT_ID="your-client-id"
export FREEE_CLIENT_SECRET="your-client-secret"
export FREEE_REDIRECT_URL="https://your-app.com/callback"
export FREEE_COMPANY_ID="12345"

# アクセストークン（OAuth2フローで取得したもの）
export FREEE_ACCESS_TOKEN="your-access-token"

# 通知設定
export GOOGLE_CHAT_WEBHOOK="https://chat.googleapis.com/v1/spaces/..."
```

#### 開発/テスト環境（エミュレータ）
```bash
export FREEE_API_URL="http://localhost:8081"
export FREEE_COMPANY_ID="1"
export FREEE_ACCESS_TOKEN="any-token"  # エミュレータは何でもOK
export GOOGLE_CHAT_WEBHOOK="..."  # オプション
```

### 実行方法

```bash
# 本番環境
go run examples/unbooked_checker_production.go

# 開発環境（エミュレータ）
FREEE_API_URL=http://localhost:8081 \
FREEE_COMPANY_ID=1 \
go run examples/unbooked_checker_production.go
```

### Cron設定例

```cron
# 平日9-18時、1時間ごとに実行
0 9-18 * * 1-5 cd /app && /usr/local/go/bin/go run examples/unbooked_checker_production.go >> /var/log/freee-checker.log 2>&1
```

### Docker化

```dockerfile
FROM golang:1.23-alpine

WORKDIR /app
COPY . .

RUN go build -o freee-checker examples/unbooked_checker_production.go

CMD ["./freee-checker"]
```

```bash
docker build -t freee-checker .
docker run --env-file .env freee-checker
```

## 📊 監視・運用

### 1. **ログ監視**
```bash
# エラーログを監視
tail -f /var/log/freee-checker.log | grep ERROR

# 未仕訳件数を監視
tail -f /var/log/freee-checker.log | grep "未仕分け明細"
```

### 2. **アラート設定**
- 未仕訳が10件以上で警告
- API障害時にSlack/Google Chat通知
- プログラムが異常終了したら通知

### 3. **メトリクス収集（推奨）**
```go
// Prometheus対応
import "github.com/prometheus/client_golang/prometheus"

var (
    unbookedCount = prometheus.NewGauge(prometheus.GaugeOpts{
        Name: "freee_unbooked_transactions_total",
        Help: "Number of unbooked transactions",
    })
)
```

## ⚠️ 本番投入前のチェックリスト

- [ ] OAuth2トークンの取得・リフレッシュ実装
- [ ] トークンの安全な保存（AWS Secrets Manager等）
- [ ] エラーハンドリングの確認
- [ ] タイムアウト設定の適切性確認
- [ ] ログローテーション設定
- [ ] アラート設定
- [ ] リトライ回数・間隔の調整
- [ ] レート制限対応（freee APIの制限確認）
- [ ] 本番環境でのテスト実行
- [ ] 監視ダッシュボード作成

## 🔐 セキュリティ考慮事項

### 1. **トークン管理**
```go
// ❌ 環境変数に直接保存（開発環境のみ）
token := os.Getenv("FREEE_ACCESS_TOKEN")

// ✅ 本番環境ではSecrets Managerを使用
token := getTokenFromSecretsManager()
```

### 2. **認証情報のローテーション**
- アクセストークンの定期更新
- リフレッシュトークンの利用
- クレデンシャルのローテーション

### 3. **アクセス制限**
- 必要最小限のスコープ（`read`のみ）
- IPアドレス制限
- ネットワークセグメンテーション

## 📈 改善の優先度

### 🔴 必須（本番投入前）
1. OAuth2のトークンリフレッシュ実装
2. エラーハンドリング強化
3. ログ・監視体制の構築

### 🟡 推奨（運用開始後）
1. メトリクス収集（Prometheus等）
2. 構造化ログ（JSON形式）
3. 分散トレーシング

### 🟢 オプション
1. Webhook通知の拡張（Slack、Teams等）
2. ダッシュボード作成（Grafana等）
3. 自動仕訳機能の追加

## 📚 参考リンク

- [freee API ドキュメント](https://developer.freee.co.jp/docs)
- [OAuth2 ベストプラクティス](https://oauth.net/2/)
- [Goの本番環境ベストプラクティス](https://github.com/golang-standards/project-layout)
