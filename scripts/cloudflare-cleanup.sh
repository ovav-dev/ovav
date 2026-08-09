#!/bin/bash
# ╔══════════════════════════════════════════════════════════════════════════════╗
# ║                      CLOUDFLARE PAGES CLEANUP SCRIPT                       ║
# ║                                                                      ║
# ║  Usage: ./cloudflare-cleanup.sh [project-name]                         ║
# ║                                                                      ║
# ║  This script deletes all deployments for a Cloudflare Pages project    ║
# ║  so the project can be deleted.                                       ║
# ║                                                                      ║
# ╚══════════════════════════════════════════════════════════════════════════════╝

set -e

# Configuration
ACCOUNT_ID=$(cat .ovav/vault/tokens/CF_ACCOUNT_ID)
API_TOKEN=$(cat .ovav/vault/tokens/CF_API_TOKEN)
PROJECT_NAME="${1:-ovav-systems}"

# Colors
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}╔══════════════════════════════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║               CLOUDFLARE PAGES CLEANUP SCRIPT                             ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════════════════════════════╝${NC}"
echo ""

# Check credentials
if [ -z "$ACCOUNT_ID" ] || [ -z "$API_TOKEN" ]; then
    echo -e "${RED}❌ ERROR: Missing Cloudflare credentials${NC}"
    echo "   Ensure .ovav/vault/tokens/CF_ACCOUNT_ID and CF_API_TOKEN exist"
    exit 1
fi

# API Helper function
api_call() {
    curl -s -X "$1" "$2" \
        -H "Authorization: Bearer ${API_TOKEN}" \
        -H "Content-Type: application/json" \
        "$3"
}

# Get total deployments
echo -e "${YELLOW}📋 Getting deployment list...${NC}"
TOTAL=$(api_call GET "https://api.cloudflare.com/client/v4/accounts/${ACCOUNT_ID}/pages/projects/${PROJECT_NAME}" | jq '.result.deployments_count // 0')

if [ "$TOTAL" = "null" ] || [ "$TOTAL" = "0" ]; then
    echo -e "${GREEN}✅ No deployments found. Project can be deleted.${NC}"
    exit 0
fi

echo -e "${YELLOW}📊 Total deployments: ${TOTAL}${NC}"
echo ""
echo -e "${RED}⚠️  WARNING: This will delete ALL ${TOTAL} deployments!${NC}"
echo ""
read -p "Continue? (yes/no): " CONFIRM

if [ "$CONFIRM" != "yes" ]; then
    echo "Cancelled."
    exit 0
fi

# Delete deployments page by page
PAGE=1
DELETED=0
ERRORS=0

while true; do
    echo -e "${BLUE}📄 Processing page ${PAGE}...${NC}"
    
    RESPONSE=$(api_call GET "https://api.cloudflare.com/client/v4/accounts/${ACCOUNT_ID}/pages/projects/${PROJECT_NAME}/deployments?per_page=50&page=${PAGE}")
    
    IDS=$(echo "$RESPONSE" | jq -r '.result[].id' 2>/dev/null)
    
    if [ -z "$IDS" ] || [ "$IDS" = "null" ]; then
        echo -e "${GREEN}✅ All deployments processed!${NC}"
        break
    fi
    
    for ID in $IDS; do
        echo -n "   Deleting: ${ID:0:8}... "
        
        RESULT=$(api_call DELETE "https://api.cloudflare.com/client/v4/accounts/${ACCOUNT_ID}/pages/projects/${PROJECT_NAME}/deployments/${ID}")
        
        if echo "$RESULT" | jq -e '.success' > /dev/null 2>&1; then
            echo -e "${GREEN}✓${NC}"
            ((DELETED++))
        else
            ERROR=$(echo "$RESULT" | jq -r '.errors[0].message // "Unknown error"' 2>/dev/null)
            echo -e "${RED}✗${NC} ($ERROR)"
            ((ERRORS++))
        fi
        
        # Rate limit protection
        sleep 0.1
    done
    
    ((PAGE++))
    
    # Safety limit
    if [ $PAGE -gt 50 ]; then
        echo -e "${YELLOW}⚠️ Reached page limit (50). Stopping.${NC}"
        break
    fi
done

echo ""
echo -e "${BLUE}═══════════════════════════════════════════════════════════════════════════════${NC}"
echo -e "${GREEN}✅ CLEANUP COMPLETE${NC}"
echo ""
echo "   Deleted: $DELETED"
echo "   Errors:  $ERRORS"
echo ""

# Try to delete project
echo -e "${YELLOW}🗑️ Attempting to delete project...${NC}"

RESULT=$(api_call DELETE "https://api.cloudflare.com/client/v4/accounts/${ACCOUNT_ID}/pages/projects/${PROJECT_NAME}")

if echo "$RESULT" | jq -e '.success' > /dev/null 2>&1; then
    echo -e "${GREEN}✅ Project deleted successfully!${NC}"
else
    ERROR=$(echo "$RESULT" | jq -r '.errors[0].message // "Unknown error"' 2>/dev/null)
    echo -e "${RED}❌ Failed to delete project: $ERROR${NC}"
    echo ""
    echo "   Manual steps required. Visit:"
    echo "   https://dash.cloudflare.com/a${ACCOUNT_ID}/pages"
fi
