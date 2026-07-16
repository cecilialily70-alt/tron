# TRON 靓号地址生成器 v15

GPU 加速的 TRON (TRC20/USDT) 靓号地址生成器。利用 NVIDIA CUDA **RTX 5090 (32 GB)** 生成海量随机私钥，通过 **libsecp256k1（Bitcoin Core C 库）** 推导地址，匹配 7 位连续相同字符或 6 个 6 / 6 个 8 的靓号。

所有私钥推导均使用受信任的 Bitcoin 核心加密库，结果 100% 正确。

命中的靓号**实时推送**到 Telegram。每 30 分钟发送一次状态报告。

---

## 匹配规则

| 优先级 | 模式 | 地址示例 |
|--------|------|----------|
| 1 | 后 7 位相同 | `Txxxxxxxxx...AAAAAAA` |
| 2 | 前 7 位相同 | `TAAAAAAA...xxxxx` |
| 3 | 连续 6 个 6 | `Txx666666xxx...` |
| 4 | 连续 6 个 8 | `Txx888888xxx...` |

---

## 架构

```
GPU cuRAND           Go 调度器             加密推导               匹配检查
(随机私钥 32B)  -->  (30 核并发)  -->  libsecp256k1 (Bitcoin)  -->  Base58 编码
                                       + Keccak-256              + 7位/666/888 匹配
                                       (受信任 C 库)              + Telegram 推送
```

GPU 不参与任何加密计算，只负责快速生成随机数。所有地址推导均使用 Go + libsecp256k1，确保私钥与地址完全匹配。

---

## 首次部署

### 1. 安装依赖

```bash
# 安装 libsecp256k1（Bitcoin 核心加密库）
sudo apt update && sudo apt install -y libsecp256k1-dev

# 确保 CUDA 和 Go 已安装
nvcc --version
go version
```

### 2. 克隆项目并编译

```bash
cd ~
git clone https://github.com/cecilialily70-alt/tron.git
cd tron
go mod tidy
make
CGO_ENABLED=1 go build -o tron-vanity main.go
```

### 3. 启动程序

```bash
tmux new -s tron
./tron-vanity -batch 134217728
```

看到启动日志后按 `Ctrl+B` 松手再按 `D` 脱离 tmux 会话，程序在后台持续运行。

### 4. 确认运行

检查 Telegram，应该已收到启动通知：

```
🚀 TRON 靓号生成器 v15

🎯 目标: 7位相同 / 6个6 / 6个8
🖥  Workers: 30 | GPU Batch: 134217728
🔒 加密: libsecp256k1 (Bitcoin C库)
```

---

## 参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-batch` | `67108864` (64M) | GPU 每批次生成的私钥数量 |
| `-token` | 已内置 | Telegram Bot Token |
| `-chat` | 已内置 | Telegram 接收消息的 Chat ID |

### 批次大小与显存占用

| 批次大小 | 显存占用 | 适用显卡 |
|----------|----------|----------|
| 33554432 (32M) | ~1 GB | RTX 3090 (24 GB) |
| 67108864 (64M) | ~2 GB | RTX 4090 / 5090 |
| 134217728 (128M) | ~4 GB | RTX 5090 (32 GB) |

```bash
# 自定义批次
./tron-vanity -batch 134217728
```

---

## 常用操作命令

### 查看运行状态

```bash
# 回到 tmux 会话看实时日志
tmux attach -t tron

# 查看 GPU 使用情况
nvidia-smi

# 查看进程是否在跑
ps aux | grep tron-vanity
```

### 停止程序

```bash
# 杀掉所有相关进程
pkill -9 tron-vanity
pkill -9 vanity_worker

# 关闭 tmux 会话
tmux kill-server
```

### 服务器重启后恢复

```bash
cd ~/tron
tmux new -s tron
./tron-vanity -batch 134217728
```

### 重新编译（拉取最新代码后）

```bash
cd ~/tron
git pull
make
CGO_ENABLED=1 go build -o tron-vanity main.go
```

### 查看 Git 提交记录

```bash
cd ~/tron
git log --oneline -5
```

---

## Telegram 消息格式

**启动通知：**
```
🚀 TRON 靓号生成器 v15

🎯 目标: 7位相同 / 6个6 / 6个8
🖥  Workers: 30 | GPU Batch: 134217728
🔒 加密: libsecp256k1 (Bitcoin C库)
```

**命中靓号（实时推送）：**
```
TNNNNNNNxxxxx...
a1b2c3d4e5f6...

🎯 TRON 靓号 (前7位相同)
```

纯文本两行：地址在上、私钥在下，Telegram 直接点击即可全选复制。

**30 分钟状态报告：**
```
📊 TRON Vanity Generator 状态报告

⏱  运行时间: 2h30m15s
🔑 已生成密钥: 45000000000
✅ 发现靓号: 3
⚡ 当前速率: 1.02 M/s
```

---

## 速率参考

| 加密引擎 | 速率 | 7 位靓号预计时间 |
|----------|------|------------------|
| libsecp256k1 (C 库) | ~1 M/s | 数小时 |

---

## 安全说明

- 私钥通过 Telegram 明文传输，仅用于学习/靓号收藏目的
- 不建议在生成的地址中存放大额资产
- 加密推导使用 Bitcoin Core 的 libsecp256k1 库，已被审计十余年
- 所有计算均在本地完成，私钥不经过任何外部服务

---

## License

MIT
