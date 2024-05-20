# 运行 WatchLog
## 确定参数配置
- LOG_PREFIX：日志前缀标识, 默认是watchlog, 支持自定义
- RUNTIME_TYPE：运行时类型，支持`docker` `containerd`
- LOGGING_OUTPUT：日志输出类型，支持主流的`kafka` `elasticsearch` `redis` `file`等

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
## 启动服务
```bash
kubectl apply -f ./kubernetes/watchlog.yaml
```

# 运行测试用例
## 前提条件
需要为每个被收集的`Controller`/`Pod`中, 注入日志采集前缀标志`watchlog_{xxx}`的环境变量, 前缀标识取决于 WatchLog 服务的环境变量 LOG_PREFIX, 默认情况下是 watchlog.
```yaml
        - env:
            - name: watchlog_default-nginx
              value: stdout
```
## 启动服务
```bash
kubectl apply -f ./kubernetes/nginx.yaml
```