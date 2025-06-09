#!/bin/bash

# This script generates central Swagger documentation for ForgeCRUD

# Terminal colors
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

echo -e "${YELLOW}Generating central Swagger documentation...${NC}"

# Go to project directory
cd "$(dirname "$0")/.." || exit

# Get the full path of the project
PROJECT_ROOT=$(pwd)
DOCS_DIR="$PROJECT_ROOT/docs"

# Install swag tool if not installed
if ! command -v swag &> /dev/null; then
    echo -e "${YELLOW}Installing swag tool...${NC}"
    go install github.com/swaggo/swag/cmd/swag@latest
fi

# Add GOPATH/bin to PATH
export PATH=$PATH:$(go env GOPATH)/bin
SWAG_CMD="swag"

# Go to API Gateway main directory
cd "$PROJECT_ROOT/api-gateway" || exit

# Generate Swagger directly through API Gateway
echo -e "${YELLOW}Generating API Gateway Swagger documentation...${NC}"

# Generate Swagger - parse all services
# Parse all dependencies in depth with --parseDepth=10 parameter
# Exclude types under shared/docs/responses with --exclude parameter
# Specify all service directories with --dir parameter
$SWAG_CMD init \
    --generalInfo ../docs/swagger.go \
    --parseDependency \
    --parseInternal \
    --parseDepth=10 \
    --exclude ".*shared/docs/responses.*" \
    --dir ../api-gateway,../auth-service,../core-service,../permission-service,../notification-service,../document-service \
    --output ../docs/swagger

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✅ Swagger documentation successfully generated${NC}"
    echo -e "${GREEN}✅ Documentation saved to: $DOCS_DIR/swagger/${NC}"
else
    echo -e "${RED}❌ Failed to generate Swagger documentation${NC}"
    # Show recent errors to see the cause of the failure
    echo -e "${YELLOW}Recent errors:${NC}"
    tail -n 20 swag-errors.log 2>/dev/null
fi

echo -e "${GREEN}Process completed!${NC}"
echo -e "${YELLOW}You can access Swagger UI at:${NC}"
echo -e "${GREEN}http://localhost:8000/swagger/index.html${NC}"