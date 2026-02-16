#!/bin/bash
# Register satellite with Ground Control, create group with images, assign to satellite
set -e

GC_URL="${GC_URL:-https://localhost:9080}"
ADMIN_PASSWORD="${ADMIN_PASSWORD:-Harbor12345}"
SAT_NAME="${SAT_NAME:-agent-satellite}"
SAT_UID="${SAT_UID:-1000}"
GROUP_NAME="${GROUP_NAME:-edge-images}"

echo "=== Registering Satellite with Ground Control ==="
echo "  GC URL:         $GC_URL"
echo "  Satellite Name: $SAT_NAME"
echo "  Workload UID:   $SAT_UID"
echo "  Group:          $GROUP_NAME"
echo ""

# Step 1: Login
echo "[1/4] Logging in..."
LOGIN_RESP=$(curl -sk -w "\n%{http_code}" -X POST "${GC_URL}/login" \
    -H "Content-Type: application/json" \
    -d "{\"username\":\"admin\",\"password\":\"${ADMIN_PASSWORD}\"}")
HTTP_CODE=$(echo "$LOGIN_RESP" | tail -1)
LOGIN_BODY=$(echo "$LOGIN_RESP" | sed '$d')

if [ "$HTTP_CODE" != "200" ]; then
    echo "ERROR: Login failed (HTTP $HTTP_CODE)"
    echo "Response: $LOGIN_BODY"
    exit 1
fi

AUTH_TOKEN=$(echo "$LOGIN_BODY" | grep -o '"token":"[^"]*"' | cut -d'"' -f4)
if [ -z "$AUTH_TOKEN" ]; then
    echo "ERROR: Failed to parse auth token"
    exit 1
fi
echo "Login successful"

# Step 2: Register satellite
echo "[2/4] Registering satellite..."
REG_RESP=$(curl -sk -w "\n%{http_code}" -X POST "${GC_URL}/api/satellites/register" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${AUTH_TOKEN}" \
    -d "{\"satellite_name\":\"${SAT_NAME}\",\"selectors\":[\"unix:uid:${SAT_UID}\"],\"attestation_method\":\"x509pop\"}")
HTTP_CODE=$(echo "$REG_RESP" | tail -1)
REG_BODY=$(echo "$REG_RESP" | sed '$d')

if [ "$HTTP_CODE" != "200" ]; then
    echo "WARNING: Registration returned HTTP $HTTP_CODE (may already exist)"
    echo "Response: $REG_BODY"
else
    echo "Satellite registered"
fi

# Step 3: Create group with nginx image
echo "[3/4] Creating group '${GROUP_NAME}' with nginx image..."
GROUP_RESP=$(curl -sk -w "\n%{http_code}" -X POST "${GC_URL}/api/groups/sync" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${AUTH_TOKEN}" \
    -d "{
        \"group\": \"${GROUP_NAME}\",
        \"registry\": \"http://10.229.209.55:8080\",
        \"artifacts\": [
            {
                \"repository\": \"library/nginx\",
                \"tag\": [\"latest\"],
                \"type\": \"image\",
                \"digest\": \"sha256:4a027e20a3f6606ecdc4a5e412ac16c636d1cdb4b390d92a8265047b6873174c\",
                \"deleted\": false
            }
        ]
    }")
HTTP_CODE=$(echo "$GROUP_RESP" | tail -1)
GROUP_BODY=$(echo "$GROUP_RESP" | sed '$d')

if [ "$HTTP_CODE" != "200" ]; then
    echo "ERROR: Group sync failed (HTTP $HTTP_CODE)"
    echo "Response: $GROUP_BODY"
    exit 1
fi
echo "Group created with nginx:latest"

# Step 4: Assign satellite to group
echo "[4/4] Assigning satellite to group..."
ASSIGN_RESP=$(curl -sk -w "\n%{http_code}" -X POST "${GC_URL}/api/groups/satellite" \
    -H "Content-Type: application/json" \
    -H "Authorization: Bearer ${AUTH_TOKEN}" \
    -d "{\"satellite\":\"${SAT_NAME}\",\"group\":\"${GROUP_NAME}\"}")
HTTP_CODE=$(echo "$ASSIGN_RESP" | tail -1)
ASSIGN_BODY=$(echo "$ASSIGN_RESP" | sed '$d')

if [ "$HTTP_CODE" != "200" ]; then
    echo "WARNING: Assignment returned HTTP $HTTP_CODE"
    echo "Response: $ASSIGN_BODY"
else
    echo "Satellite assigned to group"
fi

echo ""
echo "=== Setup Complete ==="
echo "  Satellite '${SAT_NAME}' will replicate library/nginx:latest"
echo "  On the Pi, run: ./satellite.sh"
echo "  Then: kubectl apply -f pod.yaml"
