#!/bin/bash

# Deployment script for production

set -e

ENVIRONMENT=${1:-production}
VERSION=${2:-latest}

echo "🚀 Deploying MOEX Telegram Bot ($ENVIRONMENT) v$VERSION"

# Load environment variables
if [ -f .env.$ENVIRONMENT ]; then
    echo "📝 Loading $ENVIRONMENT environment..."
    set -a
    source .env.$ENVIRONMENT
    set +a
fi

# Build Docker image
echo "🐳 Building Docker image..."
docker build -t moex-telegram-bot:$VERSION .

# Stop existing container
echo "🛑 Stopping existing container..."
docker-compose down || true

# Start new container
echo "▶️ Starting new container..."
docker-compose up -d

# Health check
echo "🏥 Performing health check..."
sleep 10

if curl -f http://localhost:8443/health &> /dev/null; then
    echo "✅ Deployment successful!"
else
    echo "❌ Deployment failed!"
    exit 1
fi

# Cleanup old images
echo "🧹 Cleaning up old images..."
docker image prune -f

echo "🎉 Deployment complete!"