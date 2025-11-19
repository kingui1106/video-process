# 新功能说明

## 功能概述

本次更新为视频流管理系统添加了以下新功能：

### 1. 摄像头管理功能
- ✅ 在Web界面添加摄像头
- ✅ 编辑现有摄像头配置
- ✅ 删除摄像头
- ✅ 启用/禁用摄像头

### 2. 智能流管理
- ✅ 自动跟踪每个流的观看者数量
- ✅ 无人观看时自动停止流（默认30秒后）
- ✅ 有人观看时自动启动流
- ✅ 节省服务器资源

### 3. 实时监控界面
- ✅ 显示所有摄像头的实时状态
- ✅ 显示每个摄像头的观看者数量
- ✅ 显示流的运行状态
- ✅ 手动启动/停止流
- ✅ 实时预览视频流

## 使用说明

### 访问界面

1. **配置管理界面**
   ```
   http://localhost:8081/config
   ```
   - 添加、编辑、删除摄像头
   - 配置ROI区域
   - 查看流地址

2. **实时监控界面**
   ```
   http://localhost:8081/monitor
   ```
   - 查看所有摄像头状态
   - 查看观看者数量
   - 手动控制流的启动/停止
   - 实时预览视频

### API接口

#### 1. 获取所有摄像头
```bash
GET /api/cameras
```

#### 2. 添加摄像头
```bash
POST /api/cameras
Content-Type: application/json

{
  "id": "camera3",
  "name": "新摄像头",
  "rtspUrl": "rtsp://192.168.1.100:554/stream",
  "enabled": true,
  "roi": []
}
```

#### 3. 更新摄像头
```bash
PUT /api/cameras/{camera_id}
Content-Type: application/json

{
  "name": "更新后的名称",
  "rtspUrl": "rtsp://192.168.1.100:554/stream",
  "enabled": true,
  "roi": []
}
```

#### 4. 删除摄像头
```bash
DELETE /api/cameras/{camera_id}
```

#### 5. 启动流
```bash
POST /api/cameras/{camera_id}/start
```

#### 6. 停止流
```bash
POST /api/cameras/{camera_id}/stop
```

#### 7. 获取所有摄像头状态
```bash
GET /api/status
```

返回示例：
```json
[
  {
    "id": "camera1",
    "name": "前门摄像头",
    "rtspUrl": "rtsp://...",
    "roi": [],
    "enabled": true,
    "isStreaming": true,
    "viewerCount": 2,
    "lastViewed": "2024-01-01T12:00:00Z"
  }
]
```

## 配置说明

### 自动停止流超时时间

在 `pkg/streamManager/streamManager.go` 中可以修改：

```go
sm := &StreamManager{
    config:      config,
    idleTimeout: 30 * time.Second, // 修改这里的时间
}
```

默认为30秒，即无人观看30秒后自动停止流。

## 配置文件

默认使用当前目录下的 `config.json` 文件。如果需要使用其他配置文件，可以通过 `-config` 参数指定：

```bash
./firescrew_multistream.exe -config my_config.json
```

## 测试步骤

1. **启动服务**
   ```bash
   # 使用默认配置文件 config.json
   ./firescrew_multistream.exe

   # 或指定配置文件
   ./firescrew_multistream.exe -config config.json
   ```

2. **访问配置界面**
   - 打开浏览器访问 `http://localhost:8081/config`
   - 点击"添加摄像头"按钮
   - 填写摄像头信息并保存

3. **访问监控界面**
   - 打开浏览器访问 `http://localhost:8081/monitor`
   - 查看摄像头状态和观看者数量
   - 点击"查看流"按钮打开视频流

4. **测试自动停止功能**
   - 打开一个视频流
   - 关闭视频流窗口
   - 等待30秒后，在监控界面查看流状态应变为"离线"

5. **测试自动启动功能**
   - 在监控界面点击"查看流"
   - 流应自动启动并显示视频

## 技术实现

### 观看者计数
- 使用 `StreamInfo` 结构跟踪每个流的观看者
- HTTP连接建立时增加计数
- HTTP连接断开时减少计数

### 自动停止机制
- 使用 `time.Timer` 实现延迟停止
- 观看者数量为0时启动定时器
- 有新观看者时取消定时器
- 定时器到期后自动停止流

### 实时状态更新
- 监控界面每2秒自动刷新状态
- 使用 `/api/status` 接口获取最新数据
- 避免页面闪烁的增量更新策略

