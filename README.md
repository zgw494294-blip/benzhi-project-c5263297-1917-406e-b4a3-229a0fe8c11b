# 舞台吊挂方案安全核验工作台

本项目面向舞台机械师、安全复核员和演出技术负责人，提供吊挂任务建档、不可变方案修订、载荷与时空冲突核验、整改闭环、两级独立复核、方案冻结和放行凭据验证。服务由 Go 原生提供 HTML、CSS、JavaScript 和同源 JSON 接口，不依赖 Node 构建链。

## 构建、运行与测试

```bash
go build ./cmd/server
go test ./...
go run ./cmd/server -addr=127.0.0.1:19081
```

也可以使用 `PORT` 端口号绑定 `127.0.0.1:<PORT>`。`-selfcheck` 会启动真实回环 HTTP 服务并自动完成一条成功业务流程：

```bash
go run ./cmd/server -selfcheck -addr=127.0.0.1:19081
```

事件快照默认保存在 `.benzhi/data`，服务只监听回环地址。
