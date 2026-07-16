#!/usr/bin/env bash
#
# setup.sh �?One-click deployment for TRON Vanity Generator
# ==========================================================
#
# Run this on your rented RTX 5090 server (Ubuntu 22.04):
#
#   1. SCP this project to the server:
#        scp -r tron-address-generator user@host:~/
#
#   2. SSH in and run:
#        cd ~/tron-address-generator && bash setup.sh
#
#   3. Answer the prompts for Telegram token and Chat ID.
#
# The script will:
#   - Install Go 1.22     (if missing)
#   - Install CUDA 13.0    (if missing, via NVIDIA repo)
#   - Build the project    (precompute + CUDA + Go binary)
#   - Start the generator  (optionally)
#
# WARNING: This script modifies system packages. Review before running.

set -euo pipefail
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

log()  { echo -e "${GREEN}[SETUP]${NC} $*"; }
warn() { echo -e "${YELLOW}[WARN]${NC}  $*"; }
err()  { echo -e "${RED}[ERROR]${NC} $*"; exit 1; }

# ================================================================
# 1. Telegram credentials (已硬编码，可通过环境变量覆盖)
# ================================================================
echo ""
echo "=============================================="
echo "  TRON Vanity Generator �?Server Setup"
echo "=============================================="
echo ""

# Defaults (hardcoded �?modify here if needed)
: "${TOKEN:=8611216521:AAGXFb_Popymx2FAi3T7VCXKOX64LRmFxHY}"
: "${CHAT:=8500753537}"

if [[ -z "$TOKEN" || -z "$CHAT" ]]; then
    err "Both TOKEN and CHAT are required."
fi

# ================================================================
# 2. Verify GPU
# ================================================================
log "Verifying GPU..."
if ! nvidia-smi &>/dev/null; then
    err "nvidia-smi not found. CUDA driver not installed?"
fi

GPU_NAME=$(nvidia-smi --query-gpu=name --format=csv,noheader 2>/dev/null | head -1)
GPU_VRAM=$(nvidia-smi --query-gpu=memory.total --format=csv,noheader 2>/dev/null | head -1)
GPU_CC=$(nvidia-smi --query-gpu=compute_cap --format=csv,noheader 2>/dev/null | head -1)

log "GPU: $GPU_NAME | VRAM: $GPU_VRAM | Compute Capability: $GPU_CC"

# ================================================================
# 3. Install Go (if missing)
# ================================================================
if command -v go &>/dev/null; then
    GO_VER=$(go version | grep -oP 'go\K[0-9]+\.[0-9]+' || echo "0.0")
    log "Go found: $(go version)"
else
    GO_VER="0.0"
fi

if [[ "${GO_VER}" < "1.21" ]]; then
    log "Installing Go 1.22.5..."
    ARCH="amd64"
    wget -q "https://go.dev/dl/go1.22.5.linux-${ARCH}.tar.gz" -O /tmp/go.tar.gz
    sudo rm -rf /usr/local/go
    sudo tar -C /usr/local -xzf /tmp/go.tar.gz
    echo 'export PATH=/usr/local/go/bin:$PATH' >> ~/.bashrc
    export PATH=/usr/local/go/bin:$PATH
    rm -f /tmp/go.tar.gz
    log "Go installed: $(go version)"
fi

# ================================================================
# 4. Install CUDA Toolkit (if missing)
# ================================================================
if command -v nvcc &>/dev/null; then
    NVCC_VER=$(nvcc --version | grep -oP 'release \K[0-9]+\.[0-9]+' || echo "0.0")
    log "nvcc found: $NVCC_VER"
else
    NVCC_VER="0.0"
fi

if [[ "${NVCC_VER}" < "12.0" ]]; then
    log "Installing CUDA Toolkit..."
    DISTRO="ubuntu2204"
    ARCH="x86_64"
    CUDA_KEY="cuda-keyring_1.1-1_all.deb"

    wget -q "https://developer.download.nvidia.com/compute/cuda/repos/${DISTRO}/${ARCH}/${CUDA_KEY}" \
         -O "/tmp/${CUDA_KEY}"
    sudo dpkg -i "/tmp/${CUDA_KEY}"
    sudo apt-get update -qq

    # Install CUDA 13.0 if available, otherwise 12.6
    if apt-cache show cuda-toolkit-13-0 &>/dev/null; then
        CUDA_PKG="cuda-toolkit-13-0"
    else
        CUDA_PKG="cuda-toolkit-12-6"
        warn "CUDA 13.0 not in repo. Installing $CUDA_PKG instead."
    fi

    sudo apt-get install -y -qq "$CUDA_PKG"

    echo 'export PATH=/usr/local/cuda/bin:${PATH}' >> ~/.bashrc
    echo 'export LD_LIBRARY_PATH=/usr/local/cuda/lib64:${LD_LIBRARY_PATH}' >> ~/.bashrc
    export PATH=/usr/local/cuda/bin:$PATH
    export LD_LIBRARY_PATH=/usr/local/cuda/lib64${LD_LIBRARY_PATH:+:$LD_LIBRARY_PATH}
    rm -f "/tmp/${CUDA_KEY}"

    if command -v nvcc &>/dev/null; then
        log "CUDA installed: $(nvcc --version | head -1)"
    else
        warn "CUDA installation may need a reboot. Try: sudo reboot"
    fi
fi

# ================================================================
# 5. Build
# ================================================================
log "Building project..."

# Download Go dependencies
log " -> go mod tidy..."
go mod tidy

# Compile CUDA GPU key generator
log " -> Compiling CUDA GPU key generator..."
ARCH=$(nvidia-smi --query-gpu=compute_cap --format=csv,noheader 2>/dev/null | head -1 | tr -d '.')
if [[ -z "$ARCH" ]]; then
    ARCH="120"  # fallback for RTX 5090
fi
nvcc -O3 -arch="sm_${ARCH}" --use_fast_math -o gpu/vanity_worker gpu/vanity.cu -lcurand

# Compile Go orchestrator
log " -> Building Go orchestrator..."
go build -o tron-vanity main.go

log "Build complete!"
echo ""

# ================================================================
# 6. Verify build
# ================================================================
log "Verifying binaries..."
if [[ -x ./tron-vanity ]]; then
    log "�?tron-vanity: $(du -h ./tron-vanity | cut -f1)"
else
    err "tron-vanity not found"
fi

if [[ -x ./gpu/vanity_worker ]]; then
    log "�?vanity_worker: $(du -h ./gpu/vanity_worker | cut -f1)"
else
    err "vanity_worker not found"
fi

# Quick GPU smoke test (3 seconds)
log "Quick GPU test (3 seconds)..."
timeout 3 ./gpu/vanity_worker --batch 262144 2>&1 | head -5 || true
echo ""

# ================================================================
# 7. Start
# ================================================================
echo ""
read -rp "Start the generator now? [Y/n]: " START
if [[ "$START" =~ ^[Nn] ]]; then
    log "Setup complete. Run manually:"
    echo "  ./tron-vanity -token \"$TOKEN\" -chat \"$CHAT\""
    echo ""
    exit 0
fi

log "Starting generator (use Ctrl+C to stop)..."
echo ""

# Use nohup to run in background with output to log file
LOGFILE="tron-vanity-$(date +%Y%m%d-%H%M%S).log"
nohup ./tron-vanity -token "$TOKEN" -chat "$CHAT" > "$LOGFILE" 2>&1 &
PID=$!

log "Generator started (PID: $PID)"
log "Log file: $LOGFILE"
log "Monitor: tail -f $LOGFILE"

# Save PID for later
echo "$PID" > tron-vanity.pid

# Send initial Telegram notification that setup succeeded
log "Done! Check your Telegram for the startup notification."
