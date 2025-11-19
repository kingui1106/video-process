# Firescrew Multistream Docker 部署指南

## 快速开始

### 1. 准备配置文件

确保 `config.json` 文件已配置好摄像头信息：

```json
{
  "webPort": ":8081",
  "cameras": [
    {
      "id": "camera1",
      "name": "前门摄像头",
      "rtspUrl": "rtsp://192.168.1.100:554/stream",
      "roi": [],
      "drawElements": [],
      "enabled": true
    }
  ]
}
```

### 2. 使用 Docker Compose 启动

```bash
# 构建并启动服务
docker-compose up -d

# 查看日志
docker-compose logs -f

# 停止服务
docker-compose down
```

### 3. 访问 Web 界面

启动后访问：
- 配置界面: http://localhost:8081/config
- 监控界面: http://localhost:8081/monitor
- 视频流: http://localhost:8081/stream/{camera_id}

## 配置说明

### 端口映射

默认映射 `8081:8081`，如需修改宿主机端口，编辑 `docker-compose.yml`:

```yaml
ports:
  - "9090:8081"  # 宿主机端口:容器端口
```

### 数据持久化

配置文件和媒体文件通过 volumes 挂载：

```yaml
volumes:
  - ./config.json:/app/config.json:rw      # 配置文件（可读写）
  - ./media:/app/media:rw                  # 媒体文件目录
  - ./assets:/app/assets:ro                # 资源文件（只读）
```

### 环境变量

可在 `docker-compose.yml` 中配置：

```yaml
environment:
  - TZ=Asia/Shanghai           # 时区设置
  - FFMPEG_LOGLEVEL=warning    # FFmpeg 日志级别
```

### 资源限制

根据摄像头数量调整资源限制：

```yaml
deploy:
  resources:
    limits:
      cpus: '2.0'      # CPU 核心数
      memory: 2G       # 内存限制
```

## 高级用法

### 仅使用 Docker（不使用 Compose）

```bash
# 构建镜像
docker build -f docker/Dockerfile.multistream -t firescrew_multistream:latest .

# 运行容器
docker run -d \
  --name firescrew_multistream \
  -p 8081:8081 \
  -v $(pwd)/config.json:/app/config.json \
  -v $(pwd)/media:/app/media \
  --restart unless-stopped \
  firescrew_multistream:latest
```

### 查看容器状态

```bash
# 查看运行状态
docker-compose ps

# 查看资源使用
docker stats firescrew_multistream

# 进入容器
docker-compose exec firescrew_multistream sh
```

### 更新配置

修改 `config.json` 后重启服务：

```bash
docker-compose restart
```

## 故障排查

### 检查日志

```bash
# 查看所有日志
docker-compose logs

# 实时查看日志
docker-compose logs -f

# 查看最近 100 行
docker-compose logs --tail=100
```

### 健康检查

服务包含健康检查，可通过以下方式查看：

```bash
docker inspect firescrew_multistream | grep -A 10 Health
```

### 常见问题

1. **端口被占用**
   - 修改 `docker-compose.yml` 中的端口映射

2. **无法连接 RTSP 流**
   - 确保 Docker 容器可以访问摄像头网络
   - 检查 RTSP URL 是否正确

3. **配置文件未生效**
   - 确保 `config.json` 挂载路径正确
   - 重启容器使配置生效

## 网络配置

### 使用主机网络模式

如果需要访问本地网络的摄像头，可使用主机网络：

```yaml
services:
  firescrew_multistream:
    network_mode: "host"
    # 移除 ports 配置，因为使用主机网络
```

### 自定义网络

```yaml
networks:
  firescrew_network:
    driver: bridge
    ipam:
      config:
        - subnet: 172.28.0.0/16
```

## 性能优化

### 多摄像头场景

- 增加 CPU 和内存限制
- 考虑使用 GPU 加速（需要 NVIDIA Docker）
- 调整 FFmpeg 参数优化性能

### 日志管理

限制日志大小：

```yaml
logging:
  driver: "json-file"
  options:
    max-size: "10m"
    max-file: "3"
```

