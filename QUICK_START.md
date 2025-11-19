# 快速开始指南

## 1. 编译程序

### Windows
```bash
go build -o firescrew_multistream.exe ./firescrew_multistream.go
```

### Linux/Mac
```bash
go build -o firescrew_multistream ./firescrew_multistream.go
```

## 2. 配置摄像头

编辑 `config_multistream.json`:

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
    }
  ]
}
```

## 3. 启动服务

### Windows
```bash
.\firescrew_multistream.exe -config config_multistream.json
```

或者直接运行:
```bash
.\start.bat
```

### Linux/Mac
```bash
./firescrew_multistream -config config_multistream.json
```

## 4. 访问服务

### Web配置界面
```
http://localhost:8080/config
```

### 视频流地址
```
http://localhost:8080/stream/camera1
http://localhost:8080/stream/camera2
```

### API接口
```bash
# 获取所有摄像头
curl http://localhost:8080/api/cameras

# 保存ROI配置
curl -X POST http://localhost:8080/api/cameras/camera1/roi \
  -H "Content-Type: application/json" \
  -d '{"roi": [{"x": 100, "y": 100, "width": 300, "height": 200}]}'
```

## 5. 客户端访问示例

### 浏览器直接访问
打开浏览器访问: `http://服务器IP:8080/stream/camera1`

### 使用curl下载
```bash
curl http://192.168.102.29:8080/stream/camera1 > stream.mjpeg
```

### 使用VLC播放
```bash
vlc http://192.168.102.29:8080/stream/camera1
```

### 嵌入到网页
```html
<img src="http://192.168.102.29:8080/stream/camera1" alt="Camera 1">
```

### 使用ffmpeg转推
```bash
ffmpeg -i http://localhost:8080/stream/camera1 \
  -c:v libx264 -preset veryfast \
  -f rtsp rtsp://target-server:8554/output
```

## 6. Web界面操作

1. **查看视频流**: 打开配置页面自动显示所有摄像头的实时画面
2. **绘制ROI区域**: 
   - 点击"绘制区域"按钮
   - 在视频画面上拖动鼠标绘制矩形
   - 可以绘制多个区域
3. **保存配置**: 点击"保存配置"按钮保存ROI设置
4. **删除区域**: 点击ROI列表中的"删除"按钮
5. **清除所有**: 点击"清除区域"按钮删除所有ROI

## 7. 客户端示例页面

打开 `client_example.html` 文件，修改服务器地址后即可在浏览器中查看所有视频流。

## 常见问题

### Q: 视频流无法显示
A: 
1. 检查RTSP URL是否正确
2. 确认FFmpeg已安装: `ffmpeg -version`
3. 检查网络连接和防火墙设置

### Q: 编译失败
A: 确保Go版本 >= 1.18，并且已安装所有依赖:
```bash
go mod tidy
```

### Q: 如何支持更多摄像头
A: 在 `config_multistream.json` 中添加更多摄像头配置即可

### Q: 如何修改端口
A: 修改配置文件中的 `webPort` 字段，例如 `":9090"`

## 架构说明

```
客户端1 ──┐
客户端2 ──┼──> HTTP :8080 ──> StreamManager ──┬──> RTSP Camera1
客户端3 ──┘                                    ├──> RTSP Camera2
                                               └──> RTSP Camera3
```

- **单端口**: 所有服务通过一个HTTP端口提供
- **URL路径区分**: 通过 `/stream/{camera_id}` 区分不同摄像头
- **多客户端**: 支持多个客户端同时访问同一视频流
- **MJPEG格式**: 兼容性好，所有浏览器都支持

## 性能优化建议

1. **降低帧率**: 修改 `select=not(mod(n\,5))` 中的数字，数字越大帧率越低
2. **降低分辨率**: 在RTSP URL中配置较低的分辨率
3. **调整JPEG质量**: 修改 `streamManager.go` 中的 `Quality: 80`
4. **限制客户端数量**: 根据服务器性能限制同时连接的客户端数量

