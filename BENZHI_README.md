# BENZHI_README

这是一个 Go 后端服务，面向深海声学浮标阵列的任务编排与遥测回收后端，使用 Go 1.26、PostgreSQL 16 和 pgx。

## 项目说明

- 项目：vance1852/ufo-observation-command
- 项目用途：面向深海声学浮标阵列的任务编排与遥测回收后端，使用 Go 1.26、PostgreSQL 16 和 pgx。系统覆盖阵列操作员登录与可撤销会话、浮标编组、观测任务审批、下潜窗口、声学分片回收、系泊交接、完整性事件、审计链和后台重试。
- Go 工具链：`golang:1.26`
- 前端工具链：无

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
./build_benzhi_docker.sh benzhi-task-295-amd64 linux/amd64
./build_benzhi_docker.sh benzhi-task-295-arm64 linux/arm64
docker run -it benzhi-task-295-amd64:latest
docker run -it --platform linux/arm64 benzhi-task-295-arm64:latest
```

## 题目验证命令

1. 预期退出码 0：`go test ./internal/worker -run '^TestWorkerFailureDoesNotCountCompletedWork0017$' -count=1`
2. 预期退出码 0：`go test ./...`
3. 预期退出码 0：`GOTOOLCHAIN=local go build -buildvcs=false ./... && GOTOOLCHAIN=local go vet ./...`
