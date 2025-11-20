#!/bin/bash
set -e

# 在运行时更新动态链接器缓存（nvidia-docker2 会在启动时挂载 NVIDIA 库）
if [ -d "/usr/local/nvidia/lib64" ]; then
    echo "Updating dynamic linker cache for NVIDIA libraries..."
    ldconfig 2>/dev/null || true
fi

# 验证 NVIDIA 库是否可加载
if ldconfig -p | grep -q libnvcuvid; then
    echo "✓ NVIDIA Video Codec SDK libraries detected"
else
    echo "⚠ Warning: libnvcuvid.so not found - GPU acceleration may not work"
    echo "   System will automatically fallback to CPU mode if GPU fails"
fi

# 执行主应用程序
exec "$@"
