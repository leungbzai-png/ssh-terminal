# SSH Terminal

一款轻量级、便携、安全的 SSH 终端，基于 Go (Wails v2) + Vue 3 + xterm.js。

## 特性

- **多标签 + 分屏**：每个分屏内独立标签栏，支持最多 4 个并排分屏
- **快速连接**：无需保存主机即可临时连接；临时密码/口令只存于内存，除非勾选"记住此主机"（此时才加密落盘）
- **导入 `~/.ssh/config`**：解析基础 OpenSSH 配置（Host/HostName/User/Port/IdentityFile），导入前预览、跳过重复项；密钥文件仅按路径引用，不复制明文私钥
- **SSH KeepAlive**：定时发送 `keepalive@openssh.com`，保持空闲连接不被中断（默认开启，间隔 30 秒，可在设置中调整）
- **SFTP 文件浏览器**：每个会话可一键打开侧边文件面板（上传/下载/新建/删除/重命名）
- **拖放上传**：从系统文件管理器直接拖入文件/文件夹，自动上传到当前 SFTP 工作目录
- **SSH 密钥管理**：内置 Ed25519 / RSA 密钥生成与管理，支持一键部署公钥到目标主机
- **命令广播栏**：底部命令栏可发送到单个标签或当前分屏的全部标签（含历史记录）
- **主题**：浅色 / 深色 / 跟随系统（xterm.js 调色板直接读取 CSS 变量）
- **便携**：所有数据（设置、主机、密钥、known_hosts）保存在 exe 同级 `data/` 目录，整个文件夹可随意移动
- **安全**：
  - 密码与私钥口令使用 AES-256-GCM 加密落盘（密钥本机随机生成）
  - 严格的 `known_hosts` 主机指纹验证，首次连接需用户确认
  - 无任何外部网络调用、无遥测、无自动更新

## 环境要求

| 工具 | 版本要求 |
|------|---------|
| Go | 1.22 及以上 |
| Node.js | 18 及以上 |
| Wails CLI | v2.12.0（构建脚本会自动安装） |
| WebView2 Runtime | Windows 11 已内置；Windows 10 首次运行 exe 时会自动提示安装 |

Go 和 Node.js 必须在 `PATH` 中可访问（直接运行 `go version` 和 `node --version` 有输出即可）。

## 构建

```bat
build-windows.bat
```

脚本会自动校验环境、首次安装 Wails CLI、编译前端与后端为单个 exe。

输出位置：`build\bin\ssh-terminal.exe`

> 该 exe 为便携式：将整个 `build\bin\` 目录（或仅 `ssh-terminal.exe`）复制到任意 Windows 机器即可运行。`data\` 子目录会在首次启动时自动创建。

## 开发模式（热重载）

```bat
dev-windows.bat
```

修改前端代码自动热刷新；修改 Go 代码自动重启。

## 项目结构

```
ssh-terminal/
├── main.go                          # Wails 入口，窗口配置
├── app.go                           # 暴露给前端的 Go API
├── go.mod
├── wails.json                       # Wails 配置（应用名、版本、图标）
├── build-windows.bat                # 一键构建
├── dev-windows.bat                  # 开发模式
├── LICENSE
├── CHANGELOG.md
├── internal/
│   ├── portable/                    # 解析 exe 同级路径（DataDir、KeysDir 等）
│   ├── config/                      # 读写 settings.json
│   ├── cryptox/                     # AES-256-GCM 加解密，管理 secret.key
│   ├── hosts/                       # 读写 hosts.json，密码加密存储
│   ├── keymgr/                      # SSH 密钥对生成、存储、索引
│   ├── sshsess/                     # SSH 会话 + PTY，known_hosts 验证
│   └── sftpx/                       # SFTP 文件操作，批量上传进度回调
└── frontend/
    ├── package.json
    ├── vite.config.ts
    └── src/
        ├── App.vue                  # 根布局，文件拖放处理
        ├── main.ts
        ├── style.css                # CSS 设计令牌 + 亮/暗主题
        ├── wails.d.ts               # Go API 的 TypeScript 类型定义
        ├── components/
        │   ├── Sidebar.vue          # 主机列表（新建/编辑/连接/删除）
        │   ├── PaneView.vue         # 单个分屏容器
        │   ├── TabBar.vue           # 标签栏（右键菜单：重连/克隆/关闭）
        │   ├── Terminal.vue         # xterm.js 终端包装（ResizeObserver，Ctrl+F 搜索）
        │   ├── SftpPanel.vue        # 侧边文件浏览器
        │   ├── CommandBar.vue       # 底部命令广播栏
        │   ├── HostDialog.vue       # 新建/编辑主机
        │   ├── SettingsDialog.vue   # 应用设置
        │   ├── KeysDialog.vue       # 密钥管理
        │   ├── HostKeyDialog.vue    # 首次连接指纹确认弹窗
        │   └── CloseConfirmDialog.vue # 关闭时活跃会话提示
        ├── composables/
        │   └── useTheme.ts          # 主题状态（亮/暗/系统）
        └── stores/
            ├── settings.ts          # Pinia：应用设置
            ├── hosts.ts             # Pinia：主机列表
            └── sessions.ts          # Pinia：分屏、标签、会话状态
```

## 依赖清单

**Go 后端：**
- `github.com/wailsapp/wails/v2` v2.12.0 — 桌面应用框架
- `golang.org/x/crypto` v0.33.0 — SSH 协议、knownhosts 解析
- `github.com/pkg/sftp` v1.13.6 — SFTP 客户端

**前端：**
- `vue` 3.x，`pinia` 2.x
- `@xterm/xterm` 5.5.0，`@xterm/addon-fit`，`@xterm/addon-search`，`@xterm/addon-web-links`
- `vite` 5.x，`typescript` 5.x

## 安全说明

- **密码存储**：使用本机随机生成的 256 位密钥（`data/secret.key`，权限 0600）对密码和私钥口令进行 AES-256-GCM 加密。密钥丢失则无法解密已存储的凭据。**最佳实践：优先使用 SSH 密钥认证，避免存储密码。**
- **主机密钥**：首次连接弹窗显示 SHA-256 指纹，确认后写入 `data/known_hosts`。**请通过带外渠道（如服务商控制台）核对指纹再接受。**
- **指纹变化**：若服务器指纹与已保存不一致，连接立即终止并报告"possible MITM"。如属正常变更（如服务器重装），请手动编辑 `data/known_hosts` 删除对应旧行。
- **网络**：程序除用户主动发起的 SSH/SFTP 连接外，不发起任何其他网络请求。

## 许可

[MIT](LICENSE)
