FROM docker.1ms.run/golang:latest AS firescrew-build-stage

# Configure Go to use China proxy for faster module downloads
ENV GOPROXY=https://goproxy.cn,https://mirrors.aliyun.com/goproxy/,direct
ENV GOSUMDB=sum.golang.google.cn

WORKDIR /build
COPY . .
RUN ls -la; go mod download
RUN mkdir /bins
RUN GOOS=linux GOARCH=amd64 go build -ldflags "-X 'main.Version=$(date +'%Y-%m-%d_%H:%M:%S')'" -o /bins/firescrew.linux.amd64
RUN GOOS=linux GOARCH=arm64 go build -ldflags "-X 'main.Version=$(date +'%Y-%m-%d_%H:%M:%S')'" -o /bins/firescrew.linux.arm64
RUN cd demoStream/rtspServer; GOOS=linux GOARCH=amd64 go build -ldflags "-X 'main.Version=$(date +'%Y-%m-%d_%H:%M:%S')'" -o /bins/rtspServer.linux.amd64
RUN cd demoStream/rtspServer; GOOS=linux GOARCH=arm64 go build -ldflags "-X 'main.Version=$(date +'%Y-%m-%d_%H:%M:%S')'" -o /bins/rtspServer.linux.arm64


FROM docker.1ms.run/debian:stable

# Configure apt to use China mirrors for faster downloads
RUN sed -i 's/deb.debian.org/mirrors.ustc.edu.cn/g' /etc/apt/sources.list.d/debian.sources && \
    sed -i 's/security.debian.org/mirrors.ustc.edu.cn/g' /etc/apt/sources.list.d/debian.sources

RUN apt-get update && apt-get install -y curl ffmpeg python3 python3-pip git netcat-openbsd

# Configure pip to use China mirrors for faster downloads
RUN pip3 config set global.index-url https://pypi.tuna.tsinghua.edu.cn/simple && \
    pip3 install ultralytics pycoral numpy Pillow --break-system-packages
COPY --from=firescrew-build-stage /bins /bins
COPY --from=firescrew-build-stage /build/docker/run-firescrew.sh /run-firescrew.sh
RUN chmod +x /run-firescrew.sh
ENTRYPOINT ["/run-firescrew.sh"]
CMD []