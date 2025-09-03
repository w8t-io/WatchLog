WatchLog
========
## 🎉 项目介绍
`WatchLog`是一个云原生容器日志采集工具。你可以使用它来收集`Docker`、`Containerd`的容器日志并发送到集中式日志管理系统中，例如`elasticsearch` `kafka` `redis`等。

## ♻️ 版本兼容

**Input**

| Service    | Version    |
|------------|------------|
| Docker     | 推荐 20.x ➕  |
| Containerd | 推荐 1.2.x ➕ |

**Output**

| Service       | Version      |
|---------------|--------------|
| Elasticsearch | 推荐 7.10.x ➕  |
| Kafka         | 推荐 2.x ➕     |
| Redis         | 推荐 6.x ➕     |

## 🚀 快速开始
### 确定参数配置
- LOG_PREFIX：日志前缀标识, 默认是watchlog, 支持自定义
- LOG_BASE_DIR：日志存储目录（挂载到WatchLog容器内的路径），默认 `/var/log/containers`
- RUNTIME_TYPE：运行时类型，支持`docker` `containerd`
- LOGGING_OUTPUT：日志输出类型，支持主流的`kafka` `elasticsearch` `redis` `file`等

**LOG_PREFIX 详细**
```yaml
            - name: LOG_PREFIX
              value: watchlog
```

**LOG_BASE_DIR 详细**
```yaml
            - name: LOG_BASE_DIR
              value: "/var/log/containers"
```

**RUNTIME_TYPE 详细**
```yaml
            - name: RUNTIME_TYPE
              value: docker
```

**LOGGING_OUTPUT 详细配置**

- kafka
```yaml
            - name: LOGGING_OUTPUT
              value: kafka
            - name: KAFKA_BROKERS
              value: 192.168.1.190:9092
```
- elasticsearch
```yaml
            - name: LOGGING_OUTPUT
              value: elasticsearch
            - name: ELASTICSEARCH_HOST
              value: "192.168.1.190"
            - name: ELASTICSEARCH_PORT
              value: "9200"
```
- redis
```yaml
            - name: LOGGING_OUTPUT
              value: redis
            - name: REDIS_HOST
              value: "192.168.1.190"
            - name: REDIS_PORT
              value: "6379"
            - name: REDIS_PASSWORD
              value: "redis@123."
```
- file
```yaml
            - name: LOGGING_OUTPUT
              value: file
            - name: FILE_PATH
              value: "/tmp/filebeat"
            - name: FILE_NAME
              value: "filebeat"
```
### 启动服务
```bash
kubectl apply -f ./deploy/kubernetes/watchlog.yaml
```

### 运行测试用例
#### 前提条件
需要为每个被收集的`Controller`/`Pod`中, 注入日志采集前缀标志`watchlog_{xxx}`的环境变量, 前缀标识取决于 WatchLog 服务的环境变量 LOG_PREFIX, 默认情况下是 watchlog.
```yaml
        - env:
            - name: watchlog_default-nginx
              value: stdout
```
#### 启动服务
```bash
kubectl apply -f ./deploy/kubernetes/nginx.yaml
```

## 🎸 支持
- 如果你觉得 WatchLog 还不错，可以通过 Star 来表示你的喜欢
- 在公司或个人项目中使用 WatchLog，并帮忙推广给伙伴使用

## 🧑‍💻 交流渠道
- [点击我](https://cairry.github.io/docs/#%E4%BA%A4%E6%B5%81%E7%BE%A4-%E8%81%94%E7%B3%BB%E6%88%91)