# 多路视频流管理系统

## 功能特性

✅ **多路视频流支持** - 支持配置和管理多个RTSP视频流  
✅ **单端口访问** - 所有视频流通过单个HTTP端口访问，使用URL路径区分  
✅ **Web配置界面** - 可视化配置界面，实时预览视频流  
✅ **ROI区域绘制** - 在Web界面上直接绘制检测区域（Region of Interest）  
✅ **实时推流** - MJPEG格式实时推送到浏览器或其他客户端  
✅ **配置持久化** - ROI配置自动保存到配置文件  

## 快速开始

### 1. 编译程序

```bash
go build -o firescrew_multistream firescrew_multistream.go
```

### 2. 配置摄像头

编辑 `config_multistream.json` 文件：

```json
{
  "webPort": ":8080",
  "cameras": [
    {
      "id": "camera1",
      "name": "前门摄像头",
      "rtspUrl": "rtsp://192.168.1.100:554/stream",
      "roi": [],
      "enabled": true
    },
    {
      "id": "camera2",
      "name": "后门摄像头",
      "rtspUrl": "rtsp://192.168.1.101:554/stream",
      "roi": [],
      "enabled": true
    }
  ]
}
```

### 3. 启动服务

```bash
./firescrew_multistream -config config_multistream.json
```

### 4. 访问Web界面

打开浏览器访问：`http://localhost:8080/config`

## API接口

### 获取所有摄像头列表

```bash
curl http://localhost:8080/api/cameras
```

### 访问视频流

每个摄像头的视频流可以通过以下URL访问：

```bash
# MJPEG流
http://localhost:8080/stream/{camera_id}

# 示例
http://localhost:8080/stream/camera1
http://localhost:8080/stream/32010000001320000999_32010000001320000123
```

### 保存ROI配置

```bash
curl -X POST http://localhost:8080/api/cameras/camera1/roi \
  -H "Content-Type: application/json" \
  -d '{
    "roi": [
      {"x": 100, "y": 100, "width": 300, "height": 200}
    ]
  }'
```

## 使用场景

### 场景1：多客户端播放

多个客户端可以同时访问同一个视频流：

```bash
# 客户端1
curl http://192.168.102.29:8080/stream/camera1

# 客户端2
curl http://192.168.102.29:8080/stream/camera1

# 浏览器访问
# 在浏览器中打开: http://192.168.102.29:8080/stream/camera1
```

### 场景2：VLC播放器

使用VLC播放器播放视频流：

```bash
vlc http://192.168.102.29:8080/stream/camera1
```

### 场景3：嵌入到网页

```html
<img src="http://192.168.102.29:8080/stream/camera1" alt="Camera 1">
```

### 场景4：使用ffmpeg转推

将视频流转推到其他RTSP服务器：

```bash
ffmpeg -i http://localhost:8080/stream/camera1 \
  -c:v libx264 -preset veryfast \
  -f rtsp rtsp://target-server:8554/output
```

## Web界面功能

### 绘制ROI区域

1. 点击"绘制区域"按钮
2. 在视频画面上拖动鼠标绘制矩形
3. 可以绘制多个区域
4. 点击"保存配置"保存ROI设置

### 管理ROI区域

- **查看区域**：所有ROI区域会以红色矩形显示在视频上
- **删除区域**：点击ROI列表中的"删除"按钮
- **清除所有**：点击"清除区域"按钮删除所有ROI

## 配置说明

### 摄像头配置项

| 字段 | 类型 | 说明 |
|------|------|------|
| id | string | 摄像头唯一标识符，用于URL路径 |
| name | string | 摄像头显示名称 |
| rtspUrl | string | RTSP视频流地址 |
| roi | array | ROI区域数组 |
| enabled | boolean | 是否启用该摄像头 |

### ROI配置项

| 字段 | 类型 | 说明 |
|------|------|------|
| x | int | 矩形左上角X坐标 |
| y | int | 矩形左上角Y坐标 |
| width | int | 矩形宽度 |
| height | int | 矩形高度 |

## 系统要求

- Go 1.18+
- FFmpeg（用于RTSP流处理）
- 支持的操作系统：Linux, macOS, Windows

## 性能优化

- 每5帧抽取1帧，减少CPU占用
- JPEG质量设置为80，平衡画质和带宽
- 支持多个客户端同时连接，无需重复解码

## 故障排除

### 视频流无法显示

1. 检查RTSP URL是否正确
2. 确认FFmpeg已安装：`ffmpeg -version`
3. 检查网络连接和防火墙设置

### ROI配置无法保存

1. 确认config.json文件有写入权限
2. 检查浏览器控制台是否有错误信息

## 许可证

与主项目相同

