#!/bin/bash
set -e

# TypeScript型定義をOpenAPI仕様から自動生成するスクリプト

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(cd "$SCRIPT_DIR/.." && pwd)"
EMULATOR_PATH="$PROJECT_ROOT/emulator"
SWAGGER_SPEC="$EMULATOR_PATH/docs/openapi/swagger.yaml"
OPENAPI_SPEC="$PROJECT_ROOT/.tmp/openapi3.yaml"
OUTPUT_DIR="$PROJECT_ROOT/shared/src/types/generated"

echo "🔄 Generating TypeScript types from OpenAPI specification..."
echo "   Swagger 2.0 spec: $SWAGGER_SPEC"
echo "   Output dir: $OUTPUT_DIR"

# Swagger 2.0仕様ファイルが存在するか確認
if [ ! -f "$SWAGGER_SPEC" ]; then
  echo "❌ Error: Swagger spec not found at $SWAGGER_SPEC"
  echo "   Please run 'cd $EMULATOR_PATH && swag init' first."
  exit 1
fi

# 出力ディレクトリを作成
mkdir -p "$OUTPUT_DIR"
mkdir -p "$PROJECT_ROOT/.tmp"

# Swagger 2.0をOpenAPI 3.0に変換
echo "🔄 Converting Swagger 2.0 to OpenAPI 3.0..."
npx swagger2openapi "$SWAGGER_SPEC" -o "$OPENAPI_SPEC"

# openapi-typescriptでTypeScript型定義を生成
echo "🔄 Generating TypeScript types from OpenAPI 3.0..."
npx openapi-typescript "$OPENAPI_SPEC" \
  --output "$OUTPUT_DIR/freee-api.ts" \
  --alphabetize \
  --path-params-as-types

echo "✅ TypeScript types generated successfully!"
echo "   File: $OUTPUT_DIR/freee-api.ts"

# 生成された型定義のサマリーを表示
echo ""
echo "📊 Generated types summary:"
grep "export interface" "$OUTPUT_DIR/freee-api.ts" | head -10
echo "   ..."
