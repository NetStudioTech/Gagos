#!/bin/bash
# GAGOS Docker Run Script
# Starts GAGOS with auto-generated password if not provided

set -e

# Generate random password if not provided
if [ -z "$GAGOS_PASSWORD" ]; then
    export GAGOS_PASSWORD=$(openssl rand -base64 12 2>/dev/null || cat /dev/urandom | tr -dc 'a-zA-Z0-9' | head -c 16)
fi

echo "========================================"
echo "  GAGOS - Lightweight DevOps Platform"
echo "========================================"
echo ""
echo "Password: $GAGOS_PASSWORD"
echo ""
echo "To retrieve later:"
echo "  docker exec gagos printenv GAGOS_PASSWORD"
echo ""
echo "========================================"
echo ""

# Change to the directory containing docker-compose.yml
cd "$(dirname "$0")/.."

docker-compose up -d

echo ""
echo "GAGOS is starting at http://localhost:${GAGOS_PORT:-8080}"
