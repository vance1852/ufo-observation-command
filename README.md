# UFO Observation Command

UFO Observation Command is a production-grade backend for registering and reviewing unidentified aerial observations, coordinating evidence recovery, and escalating safety incidents. The public product vocabulary is UFO observation; legacy internal package names retain their stable implementation identifiers while the HTTP contract and migrations expose the observation workflow.

Foundation scope is frozen in [SPEC_FREEZE.md](SPEC_FREEZE.md). This repository contains only the green foundation: no seeded defect, task branch, private verification patch, gold answer, or intake record.

面向深海声学浮标阵列的任务编排与遥测回收后端，使用 Go 1.26、PostgreSQL 16 和 pgx。系统覆盖阵列操作员登录与可撤销会话、浮标编组、观测任务审批、下潜窗口、声学分片回收、系泊交接、完整性事件、审计链和后台重试。

核心流程由“观测计划 -> 浮标编组 -> 下潜采集 -> 分片校验 -> 数据封存”和“设备租约 -> 指令下发 -> 回执确认 -> 异常隔离 -> 审计追踪”组成。阵列调度员、声学质检员与只读审计员拥有不同业务权限。

## 启动

```bash
docker compose up -d postgres
go test ./... -count=1
go run ./cmd/server
```

默认数据库连接为 `postgres://acoustic:acoustic@localhost:5432/ufo_observation_command?sslmode=disable`，可使用 `DATABASE_URL` 覆盖。`GET /healthz` 检查进程，`GET /readyz` 检查 PostgreSQL。开发账号为 `admin / admin123`，登录、当前身份、退出撤销、角色授权和业务 API 均有自动化测试覆盖。
