# TRON Vanity Address Generator �?GPU-Accelerated (RTX 5090)

GPU-accelerated TRON (TRC20/USDT) vanity address generator. Uses NVIDIA CUDA on an **RTX 5090 (32 GB)** to generate millions of keys per second, finding addresses where the **first 3 or last 3 characters are identical** (e.g., `TAAA...`, `T...111`).

Matches are sent **instantly** to your Telegram chat. Status reports every **30 minutes**.

**Key verification**: every generated address is re-derived from its private key using Go's trusted secp256k1 library to guarantee 100% correctness.

## Architecture

```
┌──────────────────────────�?    stdout pipe (52B/record)       ┌──────────────────────�?
�?  CUDA Binary             �?────────────────────────────────�?�?  Go Orchestrator     �?
�?  (gpu/vanity_worker)     �?   32B key + 20B raw hash        �?  (tron-vanity)       �?
�?                          �?                                   �?                      �?
�?  RTX 5090 GPU:           �?                                   �?  Goroutine pool:     �?
�?  �?cuRAND �?private keys �?                                   �?  �?Checker workers   �?
�?  �?secp256k1 �?pubkeys   �?                                   �?  �?Base58 encode     �?
�?  �?Keccak-256 �?hashes   �?                                   �?  �?Pattern match     �?
�?  �?Batch: 4M keys        �?                                   �?  �?Telegram send     �?
└──────────────────────────�?                                   �?  �?30-min reporter   �?
                                                                └──────────────────────�?
```

## One-Click Setup (on your rented server)

```bash
# 1. Upload project to server
scp -r tron-address-generator user@146.115.17.138:~/

# 2. SSH into server
ssh user@146.115.17.138

# 3. Run setup
cd ~/tron-address-generator
bash setup.sh
```

The script will:
- Detect your RTX 5090 automatically
- Install Go 1.22 (if needed)
- Install CUDA Toolkit (if needed)
- Build everything
- Start the generator

**Credentials used:**
- Token: `8611216521:AAGXFb_Popymx2FAi3T7VCXKOX64LRmFxHY`
- Chat ID: `8500753537`

## Manual Build

### Requirements

| Component | Version |
|-----------|---------|
| Ubuntu | 22.04 |
| Go | 1.21+ |
| CUDA Toolkit | 13.0 (or 12.6) |
| NVIDIA Driver | 550+ |

### Steps

```bash
# 1. Install dependencies (if not using setup.sh)
sudo apt update
sudo apt install golang-go -y

# For CUDA, follow NVIDIA's official guide:
# https://developer.nvidia.com/cuda-downloads

# 2. Clone the project
git clone https://github.com/huahuade/tron-address-generator.git
cd tron-address-generator

# 3. Build (auto-detects GPU architecture)
make

# 4. Run
./tron-vanity -token "8611216521:AAGXFb_Popymx2FAi3T7VCXKOX64LRmFxHY" -chat "8500753537"
```

### Custom GPU Architecture

The Makefile auto-detects your GPU via `nvidia-smi`. To override:

```bash
# RTX 5090 (Blackwell, compute capability 12.0)
make CUDA_ARCH=sm_120

# RTX 4090 (Ada Lovelace)
make CUDA_ARCH=sm_89

# H100 (Hopper)
make CUDA_ARCH=sm_90a
```

## Tuning for Your Server

Your server specs: **RTX 5090 / 32 GB VRAM / 48 CPU cores / 56 GB RAM**

| Parameter | Default | Notes |
|-----------|---------|-------|
| GPU batch | 4,194,304 (4M) | Uses ~660 MB VRAM per batch |
| CPU workers | 48 (capped) | One per allocated core |
| Channel buffer | 262,144 records | ~13 MB heap |

To adjust at runtime:

```bash
./tron-vanity \
  -token "YOUR_TOKEN" \
  -chat "YOUR_CHAT_ID" \
  -batch 8388608 \
  -workers 48
```

### VRAM Budget

| Batch Size | Device VRAM | Host RAM |
|-----------|-------------|---------|
| 1M (1<<20) | ~170 MB | ~160 MB |
| 4M (4<<20) | ~660 MB | ~640 MB |
| 8M (8<<20) | ~1.3 GB | ~1.3 GB |
| 16M (16<<20) | ~2.6 GB | ~2.5 GB |

## Output

### Telegram Messages

**Startup:**
```
🚀 TRON 3位靓号生成器已启�?

🎯 目标: �?�?�?位相�?(3位数靓号)
🖥  CPU Workers: 48
📦 GPU Batch: 4194304
�?状态报�? �?30 分钟
```

**Match found (instant):**
```
🎯 发现 TRON 靓号 (3�?�?

🔹 地址: `TXXXxXXxXXxXXxXXxXXxXXxXXxAAAAA`
🔑 私钥: `a1b2c3d4e5f6...`
📌 模式: �?�?�?位相�?
🔒 私钥已校�?
```

**30-minute report:**
```
📊 TRON Vanity Generator 状态报�?

�? 运行时间: 2h30m15s
🔑 已生成密�? 45000000000
�?发现靓号: 3
�?当前速率: 5.12 M/s
```

## Performance Expectations (RTX 5090)

| Metric | Value |
|--------|-------|
| Expected keys/sec | 5-20 M/s |
| Time to find 3-char vanity | ~2-10 minutes (avg) |
| GPU VRAM used (4M batch) | ~660 MB |
| CPU usage | ~15% of 48 cores |

## Project Structure

```
tron-address-generator/
├── main.go                      # Go orchestrator
├── go.mod                       # Go module
├── Makefile                     # Build system (auto-detects GPU)
├── setup.sh                     # One-click deployment script
├── README.md                    # This file
├── cmd/
�?  └── gen_precompute/
�?      └── main.go              # Generator for precomputed G multiples
├── gpu/
�?  ├── vanity.cu                # CUDA kernels (secp256k1, Keccak-256)
�?  └── precomputed_g.h          # Auto-generated G multiples header
├── telegram/
�?  └── telegram.go              # Telegram Bot API client
├── checker/
�?  └── checker.go               # CPU-side vanity pattern checker
└── stats/
    └── stats.go                 # Statistics + 30-min reporter
```

## Monitoring

```bash
# Watch GPU utilization
watch -n 1 nvidia-smi

# Watch application logs
tail -f tron-vanity-*.log

# Check if running
ps aux | grep tron-vanity

# Stop
kill $(cat tron-vanity.pid)
```

## Troubleshooting

### CUDA compilation errors
- Verify nvcc version: `nvcc --version`
- CUDA 13.0 may not be in NVIDIA's repo yet. Use `setup.sh` which auto-selects 12.6 as fallback.
- Confirm GPU is visible: `nvidia-smi`

### "no CUDA-capable device detected"
```bash
sudo nvidia-persistenced --user nvidia-persistenced
sudo nvidia-smi -pm 1
```

### Telegram messages not arriving
- Verify you messaged the bot with `/start` first
- Check Chat ID: message `@userinfobot` on Telegram

### Low GPU utilization
- Increase batch size: `-batch 8388608`
- The bottleneck may be PCIe bandwidth (stdout pipe). Try `-batch 16777216`.

## Security Notice

**Private keys are transmitted in plaintext via Telegram.** This is for learning/vanity purposes. Do not use generated addresses for significant real funds.

## License

MIT
