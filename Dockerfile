FROM registry.cn-hangzhou.aliyuncs.com/opsre/golang:1.21.9-alpine3.19 AS build

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY=https://goproxy.cn,direct

COPY . /workspace

WORKDIR /workspace

RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o watchlog ./main.go && \
    chmod 777 watchlog

FROM registry.js.design/library/filebeat:7.17.10_python2

COPY --from=build /workspace/watchlog /usr/share/filebeat/watchlog/watchlog

COPY assets/entrypoint assets/filebeat/ assets/healthz /usr/share/filebeat/watchlog/

RUN /usr/bin/chmod +x /usr/share/filebeat/watchlog/watchlog /usr/share/filebeat/watchlog/healthz /usr/share/filebeat/watchlog/config.filebeat

HEALTHCHECK CMD /usr/share/filebeat/healthz

WORKDIR /usr/share/filebeat/

ENV PILOT_TYPE=filebeat

ENTRYPOINT ["python2", "/usr/share/filebeat/watchlog/entrypoint"]
