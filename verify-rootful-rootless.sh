#!/bin/bash

echo "=========================================="
echo "🔍 VERIFYING ROOTFUL vs ROOTLESS SETUP"
echo "=========================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
BLUE='\033[0;34m'
NC='\033[0m' # No Color

echo -e "${BLUE}1. Checking Container Status${NC}"
echo "----------------------------------------"
docker ps --format "table {{.Names}}\t{{.Status}}" | grep api-caller

echo ""
echo -e "${BLUE}2. Checking User in ROOTFUL Container${NC}"
echo "----------------------------------------"
echo -n "User: "
docker exec api-caller-rootful whoami
echo -n "ID: "
docker exec api-caller-rootful id
echo -n "Process owner: "
docker exec api-caller-rootful ps aux | grep api-caller | head -1

echo ""
echo -e "${BLUE}3. Checking User in ROOTLESS Container${NC}"
echo "----------------------------------------"
echo -n "User: "
docker exec api-caller-rootless whoami
echo -n "ID: "
docker exec api-caller-rootless id
echo -n "Process owner: "
docker exec api-caller-rootless ps aux | grep api-caller | head -1

echo ""
echo -e "${BLUE}4. Checking File Ownership${NC}"
echo "----------------------------------------"
echo "Rootful binary:"
docker exec api-caller-rootful ls -la /root/ | grep api-caller
echo "Rootless binary:"
docker exec api-caller-rootless ls -la /home/appuser/ | grep api-caller

echo ""
echo -e "${BLUE}5. Checking Security Options${NC}"
echo "----------------------------------------"
echo "Rootful Security:"
docker inspect api-caller-rootful --format '{{.HostConfig.SecurityOpt}}'
echo "Rootless Security:"
docker inspect api-caller-rootless --format '{{.HostConfig.SecurityOpt}}'

echo ""
echo -e "${BLUE}6. Checking Capabilities${NC}"
echo "----------------------------------------"
echo "Rootful CapDrop:"
docker inspect api-caller-rootful --format '{{.HostConfig.CapDrop}}'
echo "Rootless CapDrop:"
docker inspect api-caller-rootless --format '{{.HostConfig.CapDrop}}'

echo ""
echo -e "${BLUE}7. Checking User Configuration${NC}"
echo "----------------------------------------"
echo "Rootful User Config:"
docker inspect api-caller-rootful --format '{{.Config.User}}'
echo "Rootless User Config:"
docker inspect api-caller-rootless --format '{{.Config.User}}'

echo ""
echo -e "${BLUE}8. Testing Write Permissions${NC}"
echo "----------------------------------------"
echo "Rootful write test:"
docker exec api-caller-rootful sh -c "touch /tmp/test && echo 'Write successful' || echo 'Write failed'"
docker exec api-caller-rootful rm -f /tmp/test

echo "Rootless write test to /tmp:"
docker exec api-caller-rootless sh -c "touch /tmp/test && echo 'Write successful' || echo 'Write failed'"
docker exec api-caller-rootless rm -f /tmp/test

echo "Rootless write test to root directory (should fail):"
docker exec api-caller-rootless sh -c "touch /test 2>&1 || echo 'Write correctly denied'"

echo ""
echo "=========================================="
echo -e "${GREEN}✅ VERIFICATION COMPLETE${NC}"
echo "=========================================="
echo ""
echo "KEY DIFFERENCES TO LOOK FOR:"
echo "- Rootful should run as 'root' (UID 0)"
echo "- Rootless should run as 'appuser' (UID 1000)"
echo "- Rootless should have SecurityOpt: no-new-privileges"
echo "- Rootless should have ALL capabilities dropped"
echo ""

