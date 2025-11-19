#!/bin/bash

# GPU 性能监控脚本
# 用于监控 RTX 4090 在视频流处理中的表现

echo "=== Firescrew GPU 性能监控 ==="
echo ""

# 检查容器是否运行
if ! docker ps | grep -q firescrew_multistream; then
    echo "❌ 容器未运行"
    exit 1
fi

echo "✓ 容器运行中"
echo ""

# 显示 GPU 信息
echo "📊 GPU 状态:"
nvidia-smi --query-gpu=index,name,temperature.gpu,utilization.gpu,utilization.memory,memory.used,memory.total,power.draw,power.limit --format=csv,noheader,nounits | \
while IFS=, read -r idx name temp gpu_util mem_util mem_used mem_total power_draw power_limit; do
    echo "  GPU $idx: $name"
    echo "    温度: ${temp}°C"
    echo "    GPU 利用率: ${gpu_util}%"
    echo "    显存利用率: ${mem_util}%"
    echo "    显存使用: ${mem_used}MB / ${mem_total}MB"
    echo "    功耗: ${power_draw}W / ${power_limit}W"
done
echo ""

# 显示编解码器使用情况
echo "🎬 编解码器状态:"
nvidia-smi dmon -c 1 -s u 2>/dev/null | tail -n 1 | awk '{print "  解码器: "$4"%\n  编码器: "$5"%"}'
echo ""

# 显示容器资源使用
echo "🐳 容器资源使用:"
docker stats --no-stream --format "table {{.Container}}\t{{.CPUPerc}}\t{{.MemUsage}}\t{{.NetIO}}" | grep firescrew
echo ""

# 显示流状态
echo "📹 视频流状态:"
curl -s http://localhost:8081/api/status | jq -r '.[] | "  \(.name): \(if .isStreaming then "🟢 运行中" else "⚫ 离线" end) - 观看者: \(.viewerCount)"' 2>/dev/null || echo "  无法获取流状态"
echo ""

# 性能建议
echo "💡 性能建议:"
gpu_util=$(nvidia-smi --query-gpu=utilization.gpu --format=csv,noheader,nounits)
if [ "$gpu_util" -lt 20 ]; then
    echo "  ✓ GPU 利用率较低 ($gpu_util%)，可以增加更多视频流"
elif [ "$gpu_util" -lt 60 ]; then
    echo "  ✓ GPU 利用率正常 ($gpu_util%)，性能良好"
elif [ "$gpu_util" -lt 90 ]; then
    echo "  ⚠ GPU 利用率较高 ($gpu_util%)，接近性能上限"
else
    echo "  ❌ GPU 利用率过高 ($gpu_util%)，建议减少视频流或降低分辨率"
fi

temp=$(nvidia-smi --query-gpu=temperature.gpu --format=csv,noheader,nounits)
if [ "$temp" -lt 70 ]; then
    echo "  ✓ GPU 温度正常 (${temp}°C)"
elif [ "$temp" -lt 80 ]; then
    echo "  ⚠ GPU 温度偏高 (${temp}°C)，注意散热"
else
    echo "  ❌ GPU 温度过高 (${temp}°C)，请检查散热系统"
fi

