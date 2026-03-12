#!/bin/bash
set -e

echo "=========================================="
echo "Building ZTNA Agent"
echo "=========================================="
echo ""

# Check if we're in the right directory
if [ ! -f "main.go" ]; then
    echo "❌ Error: main.go not found. Run this script from the agent directory."
    exit 1
fi

echo "Step 1: Checking Go installation..."
if ! command -v go &> /dev/null; then
    echo "❌ Error: Go is not installed."
    exit 1
fi

GO_VERSION=$(go version | awk '{print $3}')
echo "   ✓ Go found: $GO_VERSION"
echo ""

echo "Step 2: Cleaning cache..."
go clean -cache 2>/dev/null || true
echo "   ✓ Cache cleaned"
echo ""

echo "Step 3: Running go mod tidy..."
go mod tidy
echo "   ✓ Dependencies resolved"
echo ""

echo "Step 4: Downloading dependencies..."
go mod download
echo "   ✓ Dependencies downloaded"
echo ""

echo "Step 5: Building agent..."
go build -v -o ztna-agent

if [ -f "ztna-agent" ]; then
    echo "   ✓ Build successful!"
    echo ""
    echo "=========================================="
    echo "✓ Agent built successfully"
    echo "=========================================="
    echo ""
    echo "Binary location: ./ztna-agent"
    echo "Binary size: $(du -h ztna-agent | cut -f1)"
    echo ""
    echo "To run:"
    echo "  sudo ./ztna-agent --config config.yaml"
    echo ""
    echo "⚠️  Note: Agent must run as root (required for iptables and TUN interface)"
else
    echo "❌ Build failed - binary not created"
    exit 1
fi
