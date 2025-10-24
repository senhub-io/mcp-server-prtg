#!/bin/bash
set -e

# Build script for PRTG MCP Server
# Builds binaries for multiple platforms

VERSION=${VERSION:-1.0.0}
BUILD_DIR="build"
BINARY_NAME="mcp-server-prtg"

echo "Building PRTG MCP Server v${VERSION}"
echo "======================================"

mkdir -p ${BUILD_DIR}

platforms=(
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
    "windows/amd64"
)

for platform in "${platforms[@]}"; do
    GOOS=${platform%/*}
    GOARCH=${platform#*/}

    output_name="${BUILD_DIR}/${BINARY_NAME}-${GOOS}-${GOARCH}"

    if [ "$GOOS" = "windows" ]; then
        output_name="${output_name}.exe"
    fi

    echo "Building for ${GOOS}/${GOARCH}..."

    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH go build \
        -trimpath \
        -ldflags "-s -w -X main.Version=${VERSION}" \
        -o "${output_name}" \
        ./cmd/server

    echo "  ✓ ${output_name}"
done

echo ""
echo "Build complete! Binaries:"
ls -lh ${BUILD_DIR}
