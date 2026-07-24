# Roadmap

## 目标

将 Figma 中的设计信息稳定地转化为可直接进入工程的资产与实现输入，并保持
本地执行、职责分离和可扩展的处理链路。

## 规划方向

### 设计 Token 导出

- 提取颜色、排版、变量、效果和间距等结构化设计决策。
- 面向目标平台生成可维护的设计 Token 与主题代码。
- 首批目标格式：Flutter ThemeData / ColorScheme / TextTheme、CSS variables、
  Tailwind 配置，以及 SwiftUI Color / Font extensions。

### 文案资产导出

- 收集 Figma 中的 TEXT 节点及其上下文。
- 导出为 Flutter ARB、Android strings.xml 与 iOS Localizable.strings。
- 为重复、缺失语义和待翻译内容提供可追踪的处理结果。

### 实现任务树

- 将节点层级、Auto Layout、样式与组件引用编译成有依赖关系的线性任务。
- 让每个任务包含完成实现所需的布局、样式和引用信息。
- 为 AI coding agent 提供可逐项执行和确认的任务输入，减少复杂设计稿中的信息遗漏。

### 视觉资产能力演进

- 扩展可导出的资产类型与格式选项，同时保持各平台的输出规则清晰一致。
- 改进资源命名、冲突处理和导出结果反馈，降低批量接入项目时的维护成本。

## 演进原则

- 每项新能力都遵循 CLI -> daemon -> plugin action -> daemon post-processing 的链路。
- 插件只负责 Figma Plugin API 执行；本地规则、文件写入和目标平台适配由 daemon 处理。
- 新能力先定义稳定的用户输入、插件原始结果和本地输出，再增加实现。
- 优先交付可独立使用的最小闭环，避免为尚未验证的场景提前引入通用框架。
