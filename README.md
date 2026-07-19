# TRON 靓号地址生成器 v17

GPU 加速的 TRON (TRC20/USDT) 靓号地址生成器。利用 NVIDIA CUDA 生成海量随机私钥，通过 **libsecp256k1（Bitcoin Core C 库）** 推导地址，匹配后 6 位连续相同字符的靓号（例如 `Txxxxx...AAAAAA`、`Txxxxx...111111`）。

所有私钥推导均使用受信任的 Bitcoin 核心加密库，结果 100% 正确。命中的靓号**实时推送**到 Telegram，每 30 分钟发送一次状态报告。

---

## 匹配规则

| 模式 | 说明 | 地址示例 |
|------|------|----------|
| 后 6 位相同 | 任意 6 位连续相同字符 | `Txxxxxxxxx...AAAAAA` |

---

## 架构

```
GPU cuRAND           Go 调度器             加密推导               匹配检查
(随机私钥 32B)  -->  (CPU 全核并发)  -->  libsecp256k1 (C 库)  -->  Base58 编码
                                          + Keccak-256              + 6 位相同匹配
                                          (全 C 热路径)              + Telegram 推送
```

GPU 只负责快速生成随机数，所有地址推导在 CPU 端完成（C 单次调用），确保私钥与地址完全匹配。

---

## 使用流程

### 1. SSH 登录服务器

```bash
ssh root@你的服务器IP
```

### 2. 拉取最新代码

```bash
cd ~
rm -rf tron
git clone https://github.com/cecilialily70-alt/tron.git
cd tron
```

### 3. 首次部署（安装依赖）

```bash
# 修复 Windows 换行符问题（如果脚本报错）
sed -i 's/\r$//' setup.sh

# 运行一键安装脚本
bash setup.sh
```

脚本会自动安装 Go 1.22.5 和 CUDA Toolkit。

如果 CUDA 未通过脚本安装成功，手动安装：

```bash
apt-get update
apt-get install -y cuda-toolkit-12-6
```

验证环境：

```bash
nvcc --version    # 应显示 CUDA 12.6+
go version        # 应显示 go1.22+
```

### 4. 编译项目

```bash
make clean
make
```

看到 `OK tron-vanity` 即编译成功。

如 `make` 报 `go: No such file or directory`，手动编译：

```bash
go mod tidy
CGO_ENABLED=1 go build -o tron-vanity .
```

**适配不同 GPU 架构：**

| GPU | CUDA Arch | 编译命令 |
|-----|-----------|----------|
| RTX 5090 | sm_120 | `make`（默认） |
| RTX 4090 / 5070 Ti | sm_89 | `make CUDA_ARCH=sm_89` |
| RTX 3090 | sm_86 | `make CUDA_ARCH=sm_86` |

### 5. 后台启动（tmux）

```bash
# 创建名为 tron 的后台会话
tmux new -s tron

# 启动程序
./tron-vanity -batch 134217728
```

看到启动日志后，按 `Ctrl+B` 松手再按 `D` 脱离会话，程序在后台持续运行。

### 6. 监控运行状态

```bash
# 重新进入 tmux 查看实时日志
tmux attach -t tron

# 查看 GPU 负载
nvidia-smi

# 查看进程是否在跑
ps aux | grep tron-vanity
```

---

## 参数说明

| 参数 | 默认值 | 说明 |
|------|--------|------|
| `-batch` | `67108864` (64M) | GPU 每批次生成的私钥数量 |
| `-token` | 已内置 | Telegram Bot Token |
| `-chat` | 已内置 | Telegram 接收消息的 Chat ID |

### 批次大小参考

| 批次大小 | 显存占用 | 适用显卡 |
|----------|----------|----------|
| 33554432 (32M) | ~1 GB | RTX 3090 (24 GB) |
| 67108864 (64M) | ~2 GB | RTX 4090 / 5070 Ti / 5090 |
| 134217728 (128M) | ~4 GB | RTX 5090 (32 GB) |

```bash
# RTX 5090 推荐
./tron-vanity -batch 134217728

# RTX 5070 Ti / 4090 推荐
./tron-vanity -batch 67108864

# RTX 3090 推荐
./tron-vanity -batch 33554432
```

---

## 常用操作命令

### 停止程序

```bash
# 强制终止 Go 主程序和 GPU 工作进程
pkill -f tron-vanity
pkill -f vanity_worker

# 检查 GPU 是否已空闲
nvidia-smi
```

### 关闭 tmux 会话

```bash
# 查看所有 tmux 会话
tmux ls

# 关闭指定的会话
tmux kill-session -t tron

# 一次性关闭所有 tmux 会话
tmux kill-server
```

### 重启程序（服务器重启后）

```bash
cd ~/tron
tmux new -s tron
./tron-vanity -batch 134217728
```

### 拉取最新代码后重新编译

```bash
cd ~
rm -rf tron
git clone https://github.com/cecilialily70-alt/tron.git
cd tron
go mod tidy
CGO_ENABLED=1 go build -o tron-vanity .
```

### 彻底清空重新部署

```bash
cd ~
rm -rf tron
pkill -f tron-vanity
pkill -f vanity_worker
git clone https://github.com/cecilialily70-alt/tron.git
cd tron
bash setup.sh
make
```

---

## Telegram 消息格式

**启动通知：**

```
🚀 TRON 靓号生成器 v17

🎯 目标: 尾号6位相同 (任意字符)
🖥  Workers: 30 | GPU Batch: 134217728
🔒 加密: libsecp256k1 + C Keccak + C Base58
⚡ 全C热路径，单次CGo调用
```

**命中靓号（实时推送）：**

```
TNNNNNNNxxxxx...
a1b2c3d4e5f6...

🎯 TRON 靓号 (尾6位相同)
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

| 加密引擎 | 速率 | 6 位靓号预计时间 |
|----------|------|------------------|
| libsecp256k1 (C 热路径，单次 CGo) | ~1 M/s | 数小时 |

---

## 安全说明

- 私钥通过 Telegram 明文传输，仅用于学习/靓号收藏目的
- 不建议在生成的地址中存放大额资产
- 加密推导使用 Bitcoin Core 的 libsecp256k1 库，已被审计十余年
- 所有计算均在本地完成，私钥不经过任何外部服务

---

## License

MIT
