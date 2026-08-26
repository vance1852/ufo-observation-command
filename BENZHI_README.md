# BENZHI_README

这是一个 Go 后端服务，面向深海声学浮标阵列的任务编排与遥测回收后端，使用 Go 1.26、PostgreSQL 16 和 pgx。

## 标准构建、运行和测试命令

进入容器后执行：

```bash
# 编译
cd '/app' && GOTOOLCHAIN=local go build ./...

# 启动
cd '/app' && GOTOOLCHAIN=local go run ./cmd/server

# 测试
cd '/app' && GOTOOLCHAIN=local go test ./...
```

## Docker 构建和进入容器

```bash
chmod +x build_benzhi_docker.sh
./build_benzhi_docker.sh benzhi-task-301-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-task-301-arm64 linux/arm64
docker run -it benzhi-task-301-amd64:latest
docker run -it --platform linux/arm64 benzhi-task-301-arm64:latest
```
