# figma-asset

`figma-asset` 是一个通过本地 Figma 插件访问 Figma Plugin API 的 CLI 工具。支持从 Figma 节点导出 PNG 和 SVG 资产，写入 Flutter、Android、iOS、web 四种平台的目录结构。

它解决的问题是：外部命令行环境不能直接调用 `node.exportAsync()`，必须让 Figma 插件在 Figma runtime 内执行导出，再把图片 bytes 交回本地进程写文件。

## 架构

```text
figma-asset CLI
  |
  | HTTP (127.0.0.1:3849)
  v
figma-asset daemon :3849
  |
  | WebSocket ws://localhost:3849/plugin
  v
Figma Asset plugin UI
  |
  | postMessage
  v
Figma plugin main -> node.exportAsync()
```

- CLI：用户入口，按需启动 daemon，发送导出请求并打印结果。
- daemon：本地常驻服务，维护插件 WebSocket 连接，转发任务，按平台规则写入文件；启动后会一直运行，直到执行 `figma-asset stop`。
- Figma 插件：通用执行层，连接 daemon，调用 Figma Plugin API，返回 bytes + 节点名。

## 安装

### macOS / Linux

无需安装 Go 环境，一行命令安装预编译二进制：

```bash
curl -fsSL https://raw.githubusercontent.com/kuopenx/figma-asset/main/install.sh | sh
```

安装位置：

```text
~/.local/bin/figma-asset              ← 二进制
~/figma-asset-plugin/                 ← 插件目录
  ├── manifest.json
  └── plugin/
      ├── code.js
      └── ui.html
```

请确保 `~/.local/bin` 已加入 `PATH`。zsh 可以在 `~/.zshrc` 中加入：

```bash
export PATH="$HOME/.local/bin:$PATH"
```

### Windows

```powershell
irm https://raw.githubusercontent.com/kuopenx/figma-asset/main/install.ps1 | iex
```

安装到 `%LOCALAPPDATA%\figma-asset\figma-asset.exe` 和 `%USERPROFILE%\figma-asset-plugin\`。

### 从源码构建（开发者）

```bash
go build -o ~/.local/bin/figma-asset ./cmd/figma-asset
```

## 版本与升级

```bash
figma-asset version           # 打印当前版本
figma-asset upgrade --check   # 检查是否有新版本
figma-asset upgrade           # 下载并安装最新版本，原地替换后自动重启 daemon
```

## 安装 Figma 插件

安装 CLI 后，插件文件已经在 `~/figma-asset-plugin/` 目录下。

1. 打开 Figma Desktop。
2. 进入 `Plugins -> Development -> Import plugin from manifest...`。
3. 选择 `~/figma-asset-plugin/manifest.json`（用 `figma-asset plugin-path` 查看完整路径）。
4. 运行 `Figma Asset` 插件，并保持插件窗口打开。

插件窗口显示 `Connected. Waiting for task...` 时，说明已经连上本地 daemon。

## 使用 CLI 导出资产

### PNG 导出

```bash
figma-asset export png \
  --platform flutter \
  --node 2005:709 \
  --out /path/to/flutter_package/assets/images \
  --name im_group_notice_arrow_icon \
  --scales 1,2,3
```

`--name` 和 `--scales` 可选。不传 `--name` 时使用 Figma 节点名，不传 `--scales` 时使用平台推荐倍率。

各平台输出结构：

```text
# flutter
<out>/im_group_notice_arrow_icon.png
<out>/2.0x/im_group_notice_arrow_icon.png
<out>/3.0x/im_group_notice_arrow_icon.png

# android (--scales 默认 1,1.5,2,3,4)
<out>/drawable-mdpi/im_group_notice_arrow_icon.png
<out>/drawable-hdpi/im_group_notice_arrow_icon.png
<out>/drawable-xhdpi/im_group_notice_arrow_icon.png
<out>/drawable-xxhdpi/im_group_notice_arrow_icon.png
<out>/drawable-xxxhdpi/im_group_notice_arrow_icon.png

# ios (--scales 默认 1,2,3)
<out>/im_group_notice_arrow_icon.imageset/im_group_notice_arrow_icon.png
<out>/im_group_notice_arrow_icon.imageset/im_group_notice_arrow_icon@2x.png
<out>/im_group_notice_arrow_icon.imageset/im_group_notice_arrow_icon@3x.png
<out>/im_group_notice_arrow_icon.imageset/Contents.json

# web (--scales 默认 2)
<out>/im_group_notice_arrow_icon@2x.png
```

### SVG 导出

```bash
figma-asset export svg \
  --platform flutter \
  --node 2005:709 \
  --out /path/to/flutter_package/assets/images \
  --name im_group_notice_arrow_icon
```

所有平台的 SVG 输出结构相同，直接写入 `<out>/name.svg`，不创建子目录：

```text
<out>/im_group_notice_arrow_icon.svg
```

推荐使用"业务或模块命名空间 + 语义名称"的 snake_case 命名，避免 `icon.svg`、`bg.svg` 这类通用名称互相覆盖。

### 批量导出

`--node` 和 `--name` 支持逗号分隔，实现批量导出：

```bash
# 多个节点，各自用 Figma 节点名
figma-asset export png \
  --platform flutter \
  --node "257:2624,258:1001,259:307" \
  --out ./assets

# 多个节点，指定各自的文件名
figma-asset export png \
  --platform flutter \
  --node "257:2624,258:1001,259:307" \
  --name "icon_home,icon_search,icon_back" \
  --out ./assets
```

批量导出时并发执行（最多 5 个节点同时），逐个打印进度：

```text
Exporting 3 nodes...
[1/3] 257:2624: 3 files
[2/3] 258:1001: 3 files
[3/3] 259:307: 3 files

Done. 3 nodes, 9 files, 0 errors.
```

单个节点导出时保持原有输出格式（直接打印文件路径）。

`--name` 要么都不传，要么数量和 `--node` 完全一致。不传 `--name` 时每个节点使用各自的 Figma 节点名。

SVG 批量导出同理：

```bash
figma-asset export svg \
  --platform flutter \
  --node "257:2624,258:1001" \
  --name "icon_home,icon_search" \
  --out ./assets
```

## 命令

```bash
figma-asset start
```

启动 daemon 并保持常驻。打开 Figma 插件后可以先执行这个命令，让插件提前连接。

```bash
figma-asset status
```

查看 daemon 是否运行、监听地址和插件是否连接。示例：

```text
name: figma-asset
listen: http://127.0.0.1:3849
plugin: connected
pendingTasks: 0
```

```bash
figma-asset restart
```

重启 daemon；如果 daemon 没有运行，则等价于 `start`。

```bash
figma-asset export png --platform <flutter|android|ios|web> --node <node-id[,node-id,...]> --out <dir> [--name <name[,name,...]>] [--scales <1,2,3>]
figma-asset export svg --platform <flutter|android|ios|web> --node <node-id[,node-id,...]> --out <dir> [--name <name[,name,...]>] [svg-options]
```

`--node` 和 `--name` 支持逗号分隔实现批量导出。如果 daemon 没运行会先自动启动。随后等待插件连接，导出资产并按平台规则写入目录。

```bash
figma-asset version              # 打印当前版本
figma-asset upgrade [--check]    # 检查或安装最新版本
figma-asset plugin-path          # 打印插件 manifest.json 路径，用于 Figma 导入
figma-asset stop                 # 停止 daemon
```

`figma-asset daemon` 是内部命令，由 CLI 自动拉起，日常不需要手动执行。

## 连接模型

Figma 插件不能主动启动本机进程，所以只打开插件时，如果 daemon 还没运行，插件会显示：

```text
Disconnected. Reconnecting in 2s...
```

这是等待本地 daemon 的状态。执行任意会启动 daemon 的命令后，插件会在下一轮重连时变成 connected：

```bash
figma-asset start
```

daemon 一旦启动会常驻复用，除非手动执行：

```bash
figma-asset stop
```

## 故障恢复

```bash
figma-asset stop              # 停止 daemon
figma-asset restart            # 重启 daemon
cat ~/figma-asset-plugin/daemon.log    # 查看日志
lsof -i :3849                 # 检查端口占用
```

如果升级失败或二进制损坏，重新安装：

```bash
curl -fsSL https://raw.githubusercontent.com/kuopenx/figma-asset/main/install.sh | sh
figma-asset restart
```

## HTTP 接口

### `POST /v1/export/png`

请求：

```json
{
  "nodeId": "2005:709",
  "outDir": "/path/to/assets/images",
  "platform": "flutter",
  "fileName": "im_group_notice_arrow_icon",
  "scales": [1, 2, 3]
}
```

`fileName` 可选（空则用节点名），`scales` 可选（空则用平台推荐倍率）。

daemon 发送给插件的 action：

```json
{
  "id": "task_xxx",
  "version": 1,
  "action": "figma.exportNodePng",
  "payload": {
    "nodeId": "2005:709",
    "scales": [1, 2, 3],
    "contentsOnly": true
  }
}
```

插件返回 bytes + nodeName 后，由 daemon 按平台规则写入文件。插件不关心平台目录、命名规则或磁盘写入。

### `POST /v1/export/svg`

请求：

```json
{
  "nodeId": "2005:709",
  "outDir": "/path/to/assets/images",
  "platform": "flutter",
  "fileName": "im_group_notice_arrow_icon",
  "outlineText": true,
  "includeIds": false,
  "simplifyStroke": true
}
```

插件返回 SVG bytes 后，daemon 写入 `<out>/name.svg`。

## 扩展原则

新增能力时沿用这个模板：

```text
CLI command
  -> daemon HTTP operation
  -> plugin action
  -> plugin raw result
  -> daemon local post-processing
```

- operation：面向用户的能力，例如 `export.png`、`export.svg`。
- action：插件侧执行的 Figma API 动作，例如 `figma.exportNodePng`、`figma.exportNodeSvg`。
- daemon：处理平台路径、文件名、写磁盘、输出格式。
- plugin：只访问 Figma 设计稿和 Plugin API。

不要在插件里加入平台业务规则，也不要让外部直接传任意 JS 给插件执行。