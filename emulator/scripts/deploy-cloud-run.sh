#!/bin/bash
# Cloud Run デプロイスクリプト

set -e

# カラー出力
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}Cloud Run デプロイスクリプト${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""

# 環境変数のチェック
if [ -z "$PROJECT_ID" ]; then
    echo -e "${RED}ERROR: PROJECT_ID が設定されていません${NC}"
    echo "Usage: PROJECT_ID=your-project-id ./scripts/deploy-cloud-run.sh"
    exit 1
fi

# デフォルト値
REGION="${REGION:-asia-northeast1}"
SERVICE_NAME="${SERVICE_NAME:-freee-checker}"
IMAGE_NAME="gcr.io/${PROJECT_ID}/${SERVICE_NAME}"

echo -e "${YELLOW}設定:${NC}"
echo "  PROJECT_ID: $PROJECT_ID"
echo "  REGION: $REGION"
echo "  SERVICE_NAME: $SERVICE_NAME"
echo "  IMAGE_NAME: $IMAGE_NAME"
echo ""

# 1. Docker イメージのビルド
echo -e "${GREEN}[1/4] Docker イメージをビルド中...${NC}"
docker build -f Dockerfile.checker -t ${IMAGE_NAME}:latest .

# 2. GCR へプッシュ
echo -e "${GREEN}[2/4] GCR へプッシュ中...${NC}"
docker push ${IMAGE_NAME}:latest

# 3. Cloud Run にデプロイ
echo -e "${GREEN}[3/4] Cloud Run にデプロイ中...${NC}"
gcloud run deploy ${SERVICE_NAME} \
  --image ${IMAGE_NAME}:latest \
  --platform managed \
  --region ${REGION} \
  --project ${PROJECT_ID} \
  --allow-unauthenticated \
  --memory 512Mi \
  --cpu 1 \
  --timeout 300 \
  --max-instances 1 \
  --set-env-vars "FREEE_CLIENT_ID=${FREEE_CLIENT_ID},FREEE_CLIENT_SECRET=${FREEE_CLIENT_SECRET},FREEE_COMPANY_ID=${FREEE_COMPANY_ID},GOOGLE_CHAT_WEBHOOK=${GOOGLE_CHAT_WEBHOOK},FREEE_AUTO_REAUTH=true"

# サービスURLを取得
SERVICE_URL=$(gcloud run services describe ${SERVICE_NAME} \
  --platform managed \
  --region ${REGION} \
  --project ${PROJECT_ID} \
  --format 'value(status.url)')

echo ""
echo -e "${GREEN}[4/4] Cloud Scheduler ジョブを作成中...${NC}"

# Cloud Scheduler ジョブの作成（週2回: 月曜と木曜の9時）
gcloud scheduler jobs create http ${SERVICE_NAME}-job \
  --location ${REGION} \
  --schedule "0 9 * * 1,4" \
  --uri "${SERVICE_URL}/check" \
  --http-method POST \
  --oidc-service-account-email "${PROJECT_ID}@appspot.gserviceaccount.com" \
  --time-zone "Asia/Tokyo" \
  --project ${PROJECT_ID} \
  || echo -e "${YELLOW}Cloud Scheduler ジョブは既に存在します（スキップ）${NC}"

echo ""
echo -e "${GREEN}========================================${NC}"
echo -e "${GREEN}✅ デプロイ完了！${NC}"
echo -e "${GREEN}========================================${NC}"
echo ""
echo -e "サービスURL: ${SERVICE_URL}"
echo -e "ヘルスチェック: ${SERVICE_URL}/health"
echo -e "手動実行: curl -X POST ${SERVICE_URL}/check"
echo ""
echo -e "${YELLOW}次のステップ:${NC}"
echo "  1. 初回OAuth認証を実行してトークンを取得"
echo "  2. トークンをSecret Managerに保存（オプション）"
echo "  3. Cloud Scheduler で自動実行を確認"
echo ""
