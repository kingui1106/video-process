# Docker 国内镜像源配置说明

本项目已配置使用国内镜像源，加速 Docker 构建和部署。

## 已配置的镜像源

### 1. Docker Hub 镜像源
Dockerfile 中使用的基础镜像已配置国内镜像：
- `docker.1ms.run` - 1ms.run Docker 镜像加速

### 2. Alpine Linux (apk) 镜像源
- 中科大镜像：`mirrors.ustc.edu.cn`

### 3. Debian/Ubuntu (apt) 镜像源
- 中科大镜像：`mirrors.ustc.edu.cn`

### 4. Go 模块代理 (GOPROXY)
- 七牛云：`https://goproxy.cn`
- 阿里云：`https://mirrors.aliyun.com/goproxy/`

### 5. Python pip 镜像源
- 清华大学镜像：`https://pypi.tuna.tsinghua.edu.cn/simple`

## Docker Daemon 配置（可选）

如果需要全局配置 Docker 镜像加速，可以配置 Docker daemon。

### Windows (Docker Desktop)

1. 打开 Docker Desktop
2. 进入 Settings -> Docker Engine
3. 添加以下配置：

```json
{
  "registry-mirrors": [
    "https://docker.1ms.run",
    "https://docker.m.daocloud.io",
    "https://docker.nju.edu.cn",
    "https://dockerproxy.com",
    "https://hub-mirror.c.163.com",
    "https://mirror.baidubce.com"
  ]
}
```

4. 点击 "Apply & Restart"

### Linux

1. 创建或编辑 `/etc/docker/daemon.json`：

```bash
sudo mkdir -p /etc/docker
sudo cp docker/daemon.json /etc/docker/daemon.json
```

2. 重启 Docker 服务：

```bash
sudo systemctl daemon-reload
sudo systemctl restart docker
```

### macOS (Docker Desktop)

与 Windows 相同，通过 Docker Desktop 的设置界面配置。

## 验证配置

### 验证 Docker 镜像加速

```bash
docker info | grep -A 10 "Registry Mirrors"
```

### 测试构建速度

```bash
# 清理缓存后重新构建
docker-compose build --no-cache
```

## 其他国内镜像源选项

### Docker Hub 镜像源备选
- `https://docker.m.daocloud.io` - DaoCloud
- `https://docker.nju.edu.cn` - 南京大学
- `https://dockerproxy.com` - DockerProxy
- `https://hub-mirror.c.163.com` - 网易
- `https://mirror.baidubce.com` - 百度云

### Go 代理备选
- `https://goproxy.io`
- `https://athens.azurefd.net`
- `https://gocenter.io`

### pip 镜像源备选
- 阿里云：`https://mirrors.aliyun.com/pypi/simple/`
- 豆瓣：`https://pypi.douban.com/simple/`
- 中科大：`https://pypi.mirrors.ustc.edu.cn/simple/`

### apt 镜像源备选
- 阿里云：`mirrors.aliyun.com`
- 清华大学：`mirrors.tuna.tsinghua.edu.cn`
- 网易：`mirrors.163.com`

## 故障排除

### 如果镜像源不可用

1. 检查网络连接
2. 尝试其他镜像源
3. 临时移除镜像源配置，使用官方源

### 构建失败

```bash
# 清理 Docker 缓存
docker system prune -a

# 重新构建
docker-compose build --no-cache
```

## 注意事项

1. 镜像源可能会有同步延迟，最新的包可能需要等待同步
2. 某些镜像源可能会不定期维护，建议配置多个备选源
3. 企业环境可能需要配置代理或私有镜像仓库

