# AGENTS.md

## 协作目标

`figma-asset` 通过本地 Figma 插件调用 Figma Plugin API，并由本地 daemon
将结果写入目标项目。保持实现简单、职责单一，并让用户可见的行为与 README
保持一致。

## 架构约束

- 固定链路是：CLI -> daemon -> plugin UI -> plugin main。不要绕过其中任一层。
- daemon 是本地能力与文件系统边界：负责请求校验、任务转发、平台规则、命名和写文件；它可以按需启动，但必须持续运行到显式 stop。
- 插件是 Figma runtime executor：只访问 Figma 文档和 Plugin API，返回原始 bytes 或 JSON；不得写本地文件、处理平台目录或决定资源名。
- daemon 监听地址、插件桥接地址及 manifest 的允许域名属于同一协议约束。改变它们时必须同步更新 daemon、插件和 manifest，并完成手工联调。
- 不引入任意 JavaScript 执行能力。所有插件 action 必须是明确白名单，并映射到有限的 Figma Plugin API 调用。
- 不添加认证、多插件路由、复杂任务调度、配置中心或旧服务兼容层，除非需求明确改变项目边界。

## 新能力与协议变更

- 新能力遵循：CLI command -> daemon HTTP operation -> plugin action -> plugin raw result -> daemon local post-processing。
- operation 面向 CLI/HTTP；action 面向插件内的 Figma API。action payload 只能包含 Figma API 所需字段，不能泄漏 `platform`、输出目录等本地概念。
- 平台输出布局、倍率、文件命名和写入逻辑必须留在 daemon；插件不得复制这些规则。
- 改变用户可见的 CLI、HTTP 协议或输出布局时，同步更新 README；未来规划归入 `ROADMAP.md`，不要把易变功能清单复制到本文件。

## 代码与验证

- Go 改动：格式化本次修改的 Go 文件，运行 `go test ./...`，再运行 `go build -o bin/figma-asset ./cmd/figma-asset`。`bin/` 是本地构建产物，不提交。
- 插件改动：至少运行 `node --check plugin/code.js`。修改插件 UI、WebSocket、消息协议或 manifest 后，必须用 Figma Desktop 完成一次真实导出联调；插件断线重连保持固定 2 秒，且不弹全局连接状态通知。
- 改变参数校验、默认倍率、文件名规范或平台输出布局时，补充或更新 Go 单测；不要只依赖手工验证。
- 重新构建后需验证运行中 daemon 的行为时，执行 `figma-asset restart` 后再测试。

## 维护原则

- `AGENTS.md` 只放稳定的协作规则、约束和验证标准。具体能力、命令示例、接口列表与人工操作步骤以 README 和源码为事实来源。
- 保持本文件短且准确；仅在反复出现的协作或质量问题得到验证后，再增加规则。
