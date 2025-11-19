#!/bin/bash

# 多路视频流测试脚本

echo "========================================="
echo "多路视频流管理系统 - 测试脚本"
echo "========================================="
echo ""

# 检查FFmpeg
echo "[1/5] 检查FFmpeg..."
if ! command -v ffmpeg &> /dev/null; then
    echo "❌ FFmpeg未安装，请先安装FFmpeg"
    exit 1
fi
echo "✅ FFmpeg已安装"
echo ""

# 编译程序
echo "[2/5] 编译程序..."
go build -o firescrew_multistream firescrew_multistream.go
if [ $? -ne 0 ]; then
    echo "❌ 编译失败"
    exit 1
fi
echo "✅ 编译成功"
echo ""

# 检查配置文件
echo "[3/5] 检查配置文件..."
if [ ! -f "config_multistream.json" ]; then
    echo "❌ 配置文件不存在: config_multistream.json"
    exit 1
fi
echo "✅ 配置文件存在"
echo ""

# 启动服务
echo "[4/5] 启动服务..."
echo "服务将在后台运行..."
./firescrew_multistream -config config_multistream.json &
SERVER_PID=$!
echo "✅ 服务已启动 (PID: $SERVER_PID)"
echo ""

# 等待服务启动
echo "[5/5] 等待服务启动..."
sleep 3
echo ""

# 显示访问信息
echo "========================================="
echo "服务已启动！"
echo "========================================="
echo ""
echo "📺 Web配置界面:"
echo "   http://localhost:8080/config"
echo ""
echo "📡 API接口:"
echo "   获取摄像头列表: http://localhost:8080/api/cameras"
echo ""
echo "🎥 视频流地址示例:"
echo "   http://localhost:8080/stream/camera1"
echo "   http://localhost:8080/stream/camera2"
echo ""
echo "🔧 测试命令:"
echo "   # 获取摄像头列表"
echo "   curl http://localhost:8080/api/cameras"
echo ""
echo "   # 在浏览器中查看视频流"
echo "   open http://localhost:8080/stream/camera1"
echo ""
echo "   # 使用VLC播放"
echo "   vlc http://localhost:8080/stream/camera1"
echo ""
echo "========================================="
echo ""
echo "按 Ctrl+C 停止服务"
echo ""

# 等待用户中断
trap "echo ''; echo '正在停止服务...'; kill $SERVER_PID; echo '✅ 服务已停止'; exit 0" INT

# 保持脚本运行
wait $SERVER_PID

