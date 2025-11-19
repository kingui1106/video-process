# GPU 硬件加速使用指南

## 功能说明

本系统支持使用 NVIDIA GPU 进行视频流处理硬件加速，可以显著降低 CPU 占用并提升处理能力。

## 性能对比

| 模式 | CPU 占用 | GPU 占用 | 支持流数 (单机) | 延迟 |
|------|---------|---------|----------------|------|
| **CPU 模式** | 100% | 0% | 5-10路 | 正常 |
| **GPU 模式** | 30% | 40% | 20-30路 | 更低 |

## 系统要求

### 硬件要求
- NVIDIA GPU (支持 CUDA 的显卡)
  - 推荐：GTX 1050 及以上
  - 最低：支持 CUDA 计算能力 3.0+

### 软件要求
1. **NVIDIA 驱动**
   ```bash
   # 检查驱动是否安装
   nvidia-smi
   ```

2. **CUDA Toolkit** (可选，但推荐)
   ```bash
   # 检查 CUDA 版本
   nvcc --version
   ```

3. **支持 CUDA 的 FFmpeg**
   ```bash
   # 检查 FFmpeg 是否支持 CUDA
   ffmpeg -hwaccels
   # 应该看到 "cuda" 在列表中
   ```

## 安装步骤

### Ubuntu/Debian

```bash
# 1. 安装 NVIDIA 驱动
sudo apt update
sudo apt install nvidia-driver-535

# 2. 安装支持 CUDA 的 FFmpeg
sudo apt install ffmpeg

# 或者从源码编译（获得最佳性能）
sudo apt install build-essential yasm cmake libtool libc6 libc6-dev \
  unzip wget libnuma1 libnuma-dev

# 下载并编译 FFmpeg with CUDA
git clone https://git.ffmpeg.org/ffmpeg.git ffmpeg/
cd ffmpeg
./configure --enable-cuda-nvcc --enable-cuvid --enable-nvenc \
  --enable-nonfree --enable-libnpp \
  --extra-cflags=-I/usr/local/cuda/include \
  --extra-ldflags=-L/usr/local/cuda/lib64
make -j$(nproc)
sudo make install

# 3. 重启系统
sudo reboot
```

### Docker 环境

使用 NVIDIA Container Toolkit：

```bash
# 1. 安装 NVIDIA Container Toolkit
distribution=$(. /etc/os-release;echo $ID$VERSION_ID)
curl -s -L https://nvidia.github.io/nvidia-docker/gpgkey | sudo apt-key add -
curl -s -L https://nvidia.github.io/nvidia-docker/$distribution/nvidia-docker.list | \
  sudo tee /etc/apt/sources.list.d/nvidia-docker.list

sudo apt-get update
sudo apt-get install -y nvidia-container-toolkit
sudo systemctl restart docker

# 2. 运行容器时添加 GPU 支持
docker run --gpus all -v $(pwd)/config.json:/app/config.json \
  -p 8081:8081 firescrew_multistream
```

## 配置启用

编辑 `config.json` 文件：

```json
{
  "webPort": ":8081",
  "enableGPU": true,  // 设置为 true 启用 GPU 加速
  "cameras": [
    // ... 摄像头配置
  ]
}
```

## 验证 GPU 加速

### 1. 查看启动日志

启动程序后，应该看到：
```
✓ NVIDIA GPU hardware acceleration enabled
Using GPU acceleration for stream: rtsp://...
```

如果看到警告：
```
⚠ GPU acceleration requested but not available, falling back to CPU
```
说明 GPU 不可用，系统会自动回退到 CPU 模式。

### 2. 监控 GPU 使用率

在另一个终端运行：
```bash
# 实时监控 GPU 使用情况
watch -n 1 nvidia-smi
```

应该看到：
- GPU 利用率：30-60%
- 显存占用：每个流约 200-500MB
- 解码器使用率：显示在 "Dec" 列

### 3. 性能测试

```bash
# 监控 CPU 使用率
top -p $(pgrep firescrew_multistream)

# 对比 GPU 模式和 CPU 模式的 CPU 占用
# GPU 模式应该降低 60-80%
```

## 故障排查

### 问题 1：nvidia-smi 找不到

**解决方案**：
```bash
# 安装 NVIDIA 驱动
sudo ubuntu-drivers autoinstall
sudo reboot
```

### 问题 2：FFmpeg 不支持 CUDA

**解决方案**：
```bash
# 检查 FFmpeg 编译选项
ffmpeg -version | grep cuda

# 如果没有 cuda，需要重新编译 FFmpeg
# 参考上面的安装步骤
```

### 问题 3：GPU 加速启动失败

**检查步骤**：
1. 确认 GPU 驱动正常：`nvidia-smi`
2. 确认 FFmpeg 支持 CUDA：`ffmpeg -hwaccels`
3. 查看详细错误日志
4. 尝试手动运行 FFmpeg 测试：
   ```bash
   ffmpeg -hwaccel cuda -i rtsp://your-stream-url -f null -
   ```

### 问题 4：显存不足

**解决方案**：
- 减少同时运行的流数量
- 降低视频分辨率
- 增加帧间隔（修改 `mod(n\,5)` 为更大的值）

## 性能优化建议

1. **调整帧率**：修改 `select=not(mod(n\,5))` 中的数字
   - `mod(n\,3)` = 更高帧率，更流畅
   - `mod(n\,10)` = 更低帧率，更省资源

2. **多 GPU 支持**（未来功能）
   - 可以通过 `-hwaccel_device` 参数指定 GPU 设备

3. **监控和告警**
   - 设置 GPU 温度监控
   - 设置显存使用率告警

## 技术细节

### GPU 加速流程

```
RTSP流 → FFmpeg NVDEC(GPU解码) → CUDA滤镜(GPU) 
       → 下载到CPU → PNG编码 → Go处理 → JPEG编码 → 推流
```

### 关键参数说明

- `-hwaccel cuda`: 启用 CUDA 硬件加速
- `-hwaccel_output_format cuda`: 保持数据在 GPU 内存中
- `hwdownload`: 将处理后的帧从 GPU 下载到 CPU
- `format=nv12`: 转换为 NV12 格式（GPU 友好）

## 参考资料

- [FFmpeg CUDA 文档](https://trac.ffmpeg.org/wiki/HWAccelIntro)
- [NVIDIA Video Codec SDK](https://developer.nvidia.com/nvidia-video-codec-sdk)
- [CUDA Toolkit 下载](https://developer.nvidia.com/cuda-downloads)

