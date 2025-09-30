#!/bin/bash

echo "🧪 Testing Metric Harvester with Rootful and Rootless Containers"
echo "================================================================="

# Function to check if container is running
check_container() {
    local container_name=$1
    if docker ps --format "table {{.Names}}" | grep -q "^${container_name}$"; then
        echo "✅ Container '$container_name' is running"
        return 0
    else
        echo "❌ Container '$container_name' is not running"
        return 1
    fi
}

# Function to test metrics collection
test_metrics() {
    local container_name=$1
    local container_type=$2
    
    echo "📊 Testing metrics for $container_type container: $container_name"
    
    # Test docker stats command
    echo "Docker stats output:"
    docker stats --no-stream --format "table {{.Container}}\\t{{.CPUPerc}}\\t{{.MemUsage}}\\t{{.NetIO}}\\t{{.BlockIO}}" $container_name
    
    # Test metric harvester endpoint
    echo "Metric harvester metrics:"
    curl -s http://localhost:8080/metrics | grep "container_.*{$container_name}" | head -5
    echo ""
}

# Start rootful container
echo "🐳 Starting rootful container..."
docker run -d --name artisan-agent-api-rootful nginx:alpine
sleep 2

# Check if rootful container is running
if check_container "artisan-agent-api-rootful"; then
    test_metrics "artisan-agent-api-rootful" "rootful"
fi

# Test with rootless container (if available)
echo "🔒 Testing rootless container..."
if command -v dockerd-rootless.sh &> /dev/null; then
    # Start rootless daemon if not running
    if ! pgrep -f "dockerd-rootless" > /dev/null; then
        echo "Starting rootless Docker daemon..."
        dockerd-rootless.sh &
        sleep 5
    fi
    
    # Run container in rootless mode
    docker run -d --name artisan-agent-api-rootless nginx:alpine
    sleep 2
    
    if check_container "artisan-agent-api-rootless"; then
        test_metrics "artisan-agent-api-rootless" "rootless"
    fi
else
    echo "⚠️  Rootless Docker not available. Install with: curl -fsSL https://get.docker.com/rootless | sh"
fi

# Cleanup
echo "🧹 Cleaning up test containers..."
docker stop artisan-agent-api-rootful artisan-agent-api-rootless 2>/dev/null || true
docker rm artisan-agent-api-rootful artisan-agent-api-rootless 2>/dev/null || true

echo "✅ Test completed!"
