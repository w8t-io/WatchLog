FROM registry.js.design/base/golang:1.18-alpine3.16 AS build

ENV GO111MODULE=on \
    CGO_ENABLED=0 \
    GOOS=linux \
    GOARCH=amd64 \
    GOPROXY=https://goproxy.cn,direct

COPY . /workspace

WORKDIR /workspace

RUN go mod tidy && \
    CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o point ./main.go && \
    chmod 777 point

FROM elastic/filebeat:6.8.23

USER root

COPY --from=build /workspace/point /usr/share/filebeat/point/point

COPY assets/entrypoint assets/filebeat/ assets/healthz /usr/share/filebeat/point/

RUN /usr/bin/chmod +x /usr/share/filebeat/point/point /usr/share/filebeat/point/healthz /usr/share/filebeat/point/config.filebeat

HEALTHCHECK CMD /usr/share/filebeat/healthz

VOLUME /var/log/filebeat

VOLUME /var/lib/filebeat

WORKDIR /usr/share/filebeat/

ENV PILOT_TYPE=filebeat

ENTRYPOINT ["python2", "/usr/share/filebeat/point/entrypoint"]
