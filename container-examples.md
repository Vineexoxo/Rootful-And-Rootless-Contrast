# Rootful vs Rootless Container Examples

## 1. Rootful Containers (Default Docker)

### Start a rootful container:
```bash
# Standard Docker (runs as root)
docker run -d --name artisan-agent-api nginx:alpine

# Check if it's running
docker ps

# Get stats
docker stats --no-stream artisan-agent-api
```

### Rootful Container Characteristics:
- ✅ Runs with root privileges
- ✅ Full access to host resources
- ✅ Standard Docker daemon
- ✅ Your metric harvester works perfectly

## 2. Rootless Containers

### Install Rootless Docker:
```bash
# Install rootless Docker
curl -fsSL https://get.docker.com/rootless | sh

# Add to PATH
export PATH=$HOME/bin:$PATH

# Start rootless daemon
dockerd-rootless.sh &
```

### Start a rootless container:
```bash
# Run container in rootless mode
docker run -d --name artisan-agent-api-rootless nginx:alpine

# Check if it's running
docker ps

# Get stats
docker stats --no-stream artisan-agent-api-rootless
```

### Rootless Container Characteristics:
- 🔒 Runs without root privileges
- 🔒 Limited access to host resources
- 🔒 Separate Docker daemon
- ✅ Your metric harvester should work (same API)

## 3. Testing Your Metric Harvester

### Update your configuration to test both:
```json
{
  "containers": {
    "monitored_names": ["artisan-agent-api", "artisan-agent-api-rootless"],
    "ignored_names": ["grafana", "prometheus", "metric-harvester"]
  }
}
```

### Run the test script:
```bash
./test-containers.sh
```

## 4. Key Differences

| Aspect | Rootful | Rootless |
|--------|---------|----------|
| **Privileges** | Root | User |
| **Security** | Less secure | More secure |
| **Resource Access** | Full | Limited |
| **Docker Socket** | `/var/run/docker.sock` | `$XDG_RUNTIME_DIR/docker.sock` |
| **Port Binding** | Any port | High ports only |
| **Volume Mounting** | Any path | User-owned paths |

## 5. Your Code Compatibility

✅ **Your code is compatible with both!** Here's why:

1. **Same Docker API**: Both use identical `docker stats` command
2. **Same Output Format**: Identical parsing logic works for both
3. **No Root Requirements**: `docker stats` doesn't need root privileges
4. **Same Metrics**: Both expose the same resource metrics

## 6. Potential Considerations

### For Rootless Containers:
- **Socket Path**: May need to mount different socket path
- **Container Names**: Might have different naming conventions
- **Resource Limits**: May have different resource visibility

### Docker Compose Update for Rootless:
```yaml
services:
  metric-harvester:
    volumes:
      # For rootless Docker
      - $XDG_RUNTIME_DIR/docker.sock:/var/run/docker.sock:ro
      # OR for rootful Docker
      - /var/run/docker.sock:/var/run/docker.sock:ro
```

## 7. Quick Test Commands

```bash
# Test rootful container
docker run -d --name test-rootful nginx:alpine
docker stats --no-stream test-rootful

# Test rootless container (if installed)
docker run -d --name test-rootless nginx:alpine
docker stats --no-stream test-rootless

# Clean up
docker stop test-rootful test-rootless
docker rm test-rootful test-rootless
```
