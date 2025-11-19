# GPU加速配置指南

## 问题说明

如果您看到以下错误：
```
[h264 @ 0x...] Cannot load libnvcuvid.so.1
[h264 @ 0x...] Failed loading nvcuvid.
[h264 @ 0x...] Failed setup for format cuda: hwaccel initialisation returned error.
```

这表示 Docker 容器中缺少 NVIDIA Video Codec SDK 库（`libnvcuvid.so.1`）。

## 解决方案

### 方案一：正确配置 NVIDIA Container Toolkit（推荐）

#### 1. 安装 NVIDIA 驱动

确保宿主机安装了 NVIDIA 驱动（版本 >= 470.x）：

```bash
# 检查驱动版本
nvidia-smi

# 如未安装，安装驱动（Ubuntu/Debian）
sudo apt-get update
sudo apt-get install -y nvidia-driver-525  # 或更新版本
sudo reboot
```

#### 2. 安装 NVIDIA Container Toolkit

```bash
# 添加 NVIDIA 仓库
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -
curl -s -L https://nvidia.github.io/nvidia-docker/$distribution/nvidia-docker.list | \
    sudo tee /etc/apt/sources.list.d/nvidia-docker.list

# 安装 nvidia-docker2
sudo apt-get update
sudo apt-get install -y nvidia-docker2

# 重启 Docker 服务
sudo systemctl restart docker
```

#### 3. 验证 GPU 支持

```bash
# 测试 NVIDIA 运行时
docker run --rm --gpus all nvidia/cuda:12.4.1-base-ubuntu22.04 nvidia-smi

# 应该能看到 GPU 信息
```

#### 4. 使用正确的 Docker Compose 配置

确保 `docker-compose.gpu.yml` 中包含 GPU 配置：

```yaml
services:
  firescrew_multistream_gpu:
    deploy:
      resources:
        reservations:
          devices:
            - driver: nvidia
              count: all
              capabilities: [gpu, video, compute]
```

#### 5. 重新构建和运行

```bash
# 重新构建镜像
docker-compose -f docker-compose.gpu.yml build

# 启动服务
docker-compose -f docker-compose.gpu.yml up -d

# 查看日志
docker-compose -f docker-compose.gpu.yml logs -f
```

### 方案二：自动回退到 CPU 模式（已实现）

系统已经实现了智能回退机制：

- 当检测到 GPU 初始化失败（连续 3 次错误）
- 自动切换到 CPU 模式继续处理
- 日志会显示回退信息

**优点**：
- 无需修改配置即可运行
- 在 GPU 不可用时自动使用 CPU

**缺点**：
- CPU 处理速度较慢
- 多路流可能导致 CPU 过载

### 方案三：禁用 GPU 加速

如果不需要 GPU 加速，可以在 `config.json` 中禁用：

```json
{
  "enableGPU": false,
  "cameras": [...]
}
```

## GPU 支持检查清单

使用以下命令检查 GPU 支持状态：

```bash
# 1. 检查宿主机驱动
nvidia-smi

# 2. 检查 Docker 中的 GPU 可见性
docker run --rm --gpus all nvidia/cuda:12.4.1-base-ubuntu22.04 nvidia-smi

# 3. 检查容器中的 FFmpeg GPU 支持
docker exec firescrew_multistream_gpu ffmpeg -hwaccels

# 应该看到 "cuda" 在列表中
```

## 常见问题

### Q: 为什么 `ffmpeg -hwaccels` 不显示 cuda？

**A**: Ubuntu 22.04 仓库中的 FFmpeg 4.4.2 默认不包含 CUDA 支持。有两种解决方案：

1. **使用 NVIDIA 运行时挂载库**（推荐）：
   - 安装 nvidia-docker2
   - 宿主机会自动将 CUDA 库挂载到容器

2. **使用支持 CUDA 的 FFmpeg**：
   - 从源码编译 FFmpeg 并启用 CUDA
   - 或使用预编译的支持 CUDA 的 FFmpeg

### Q: 容器中找不到 libnvcuvid.so.1？

**A**: 这个库由 NVIDIA 驱动提供，需要：

1. 宿主机安装完整的 NVIDIA 驱动
2. 使用 nvidia-docker2 运行容器
3. Docker Compose 配置中包含 `--gpus all`

### Q: GPU 内存不足怎么办？

**A**: 可以：

1. 减少同时处理的流数量
2. 降低视频分辨率
3. 增加帧采样间隔（当前是每 5 帧取 1 帧）
4. 回退到 CPU 模式

## 性能对比

| 模式 | 单流 CPU 使用率 | 多流 (4路) CPU 使用率 | 备注 |
|------|----------------|---------------------|------|
| GPU 加速 | ~5-10% | ~20-30% | 需要 GPU 支持 |
| CPU 模式 | ~30-50% | ~100%+ | 可能导致丢帧 |

## 日志示例

### GPU 成功启用：
```
✓ NVIDIA GPU hardware acceleration enabled
Using GPU acceleration for stream: rtsp://...
```

### GPU 初始化失败并自动回退：
```
Error from camera xxx: Cannot load libnvcuvid.so.1
⚠ GPU acceleration failed 3 times for camera xxx, falling back to CPU mode
⚠ GPU Error: libnvcuvid.so.1 not found. Please ensure:
   1. NVIDIA driver >= 470.x is installed on host
   2. nvidia-docker2 and nvidia-container-toolkit are installed
   3. Docker is configured with NVIDIA runtime
Using CPU mode for stream: rtsp://...
```

## 联系支持

如果按照上述步骤仍无法解决问题，请提供：

1. `nvidia-smi` 输出
2. `docker info | grep -i nvidia` 输出
3. 容器日志：`docker-compose -f docker-compose.gpu.yml logs`
4. FFmpeg 硬件加速列表：`docker exec container_name ffmpeg -hwaccels`
