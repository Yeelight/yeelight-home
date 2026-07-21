# yeelight-home

默认文档语言是英文：[README.md](README.md)。本文是中文使用说明。

## 官方仓库与国内镜像

[GitHub](https://github.com/Yeelight/yeelight-home) 是 Issue、贡献、CI 和发布的
规范源。国内无法稳定访问 GitHub 时，可使用只读的
[Gitee 镜像](https://gitee.com/yeelight/yeelight-home) 或
[GitCode 镜像](https://gitcode.com/Yeelight/yeelight-home)；
[GitLab.com](https://gitlab.com/Yeelight/yeelight-home) 是额外的全球备用源。
可以从任一可访问平台克隆代码，但请仍在 GitHub 提交 Issue 和贡献修改。

`yeelight-home` 是 Yeelight 智能家居 Skill 和自动化脚本使用的本地 Runtime CLI。它运行在用户自己的电脑或服务器上，本地保存凭据，解析智能家居请求，执行受支持的 Yeelight 家庭能力，并返回经过脱敏的结构化结果。

Runtime 不会被打包进 Skill。Skill 只通过 `YEELIGHT_HOME_BIN` 或 `PATH` 找到公开安装的 `yeelight-home`，然后向 `yeelight-home invoke --stdin` 发送一个 JSON 请求。

## 用普通人的话看懂整个体系

你只需要安装一个底座：**Yeelight Home**。其他项目是 AI 使用易来的不同方式，
不是彼此竞争的替代品。

| 名称 | 用普通人的话解释 | 易来对应项目 | 依赖谁 |
| --- | --- | --- | --- |
| CLI / Runtime | 装在电脑里的易来程序。它负责扫码、记住当前家庭、安全执行操作；人和脚本也能直接使用。 | **`yeelight-home`** | 不依赖矩阵里的其他项目 |
| Skill | 给 AI 的“易来使用说明书”：包含家庭规则、照明经验、安全步骤和最佳实践。 | **`yeelight-smart-home`** | 依赖 `yeelight-home` |
| MCP | 给不能安装 Skill 的 AI 客户端使用的标准云端连接方式。一套 **Yeelight MCP** 同时包含家庭理解与管理、实时状态与控制。 | **`yeelight-metadata-mcp`** + **`yeelight-iot-mcp`** | 由 `yeelight-home` 一次配置；两个云端服务执行请求时都不依赖 Runtime |

**大多数人这样选：**先安装 Yeelight Home，再让 setup 添加 Smart Home Skill。
只有 AI 客户端不能安装 Skill 时才选 MCP；写脚本、排障或明确想用终端时，
再直接使用 CLI 工作台。

## 三条上手路线，一个 CLI

普通用户不需要先分清 CLI、Skill 和 MCP。只需安装 `yeelight-home`，再选择目标：

| 路线 | 你会得到什么 | 适合谁 |
| --- | --- | --- |
| 完整智能模式（推荐） | `yeelight-smart-home` Skill 带着易来的家庭规则、照明经验和安全边界，通过 `yeelight-home` 执行。 | 希望直接用日常语言控制、管理和设计家庭的用户。 |
| 轻量连接模式 | 一次运行 `yeelight-home setup --mode mcp --mcp-source cloud`，通过本机凭据代理配置完整 Yeelight MCP 云端能力，客户端配置中不保存 Authorization。 | 客户端支持 MCP，但不方便安装 Skill 的用户。 |
| CLI 工作台 | 在终端按名称选择家庭、房间、设备和情景，或使用稳定资源命令与 `invoke --stdin`。 | 极客用户、脚本、CI 和排障人员。 |

三条路线共用本地 profile 和 Yeelight Pro APP 扫码登录体验。Skill 与本地 MCP 通过 Runtime 执行；云端 Yeelight MCP 由两个独立服务直接连接 Yeelight PRO 云。Cloud 与家庭网关 LAN 是不同执行路线，不是互相冲突的产品。

## 主要特性

- Yeelight 家庭能力，覆盖家庭、房间、区域、设备、灯组、网关、面板、旋钮、情景、自动化、诊断、灯光设计、产品知识、本地记忆和推荐。
- 产品百科搜索：支持按产品名、型号、SKU/SPU、物料编码、条码或模糊关键词查询产品资料、附件、说明书候选地址和 FAQ 候选地址。
- 薄执行模型：持久化写入、删除和高风险操作通过 Runtime 校验后直接执行；调用方负责用户确认，需要预览时使用 dry-run。
- 凭据本地化：token 优先进入系统凭据存储；普通 profile 配置只保存 region、houseId、qrDevice 等非密钥元数据。
- 多 profile：可为不同账号、区域、家庭或测试环境维护独立配置。
- 默认区域是 `cn`，也支持 `sg`、`us`、`eu`，开发场景可显式使用 `dev`。
- 输出默认脱敏，适合 Skill host、脚本和诊断工具消费。
- 本地偏好记忆和推荐反馈存储在 Runtime 数据目录中，不写入 Skill prompt。
- 同时提供面向人的资源命令和面向 Skill 的稳定 `invoke --stdin` 协议。
- `yeelight-home setup` 用中文或英文完成扫码、Skill/MCP 客户端安装和只读验证；MCP 支持自动探测、多个客户端和全部已验证客户端。
- TTY 无参数启动交互工作台；`yeelight-home menu` 可显式进入，非 TTY 无参数保持稳定帮助输出。
- `yeelight-home mcp serve --stdio` 把同一 Runtime 暴露给本机 MCP 客户端，客户端配置不保存 Yeelight Authorization。
- `cloud`、`local-preferred`、`local-only` 可让同一条命令安全选择云端或家庭网关 LAN；不确定写结果不会盲目改走云端重试。
- 使用 GoReleaser 发布 GitHub Releases、Homebrew、Scoop、npm、Linux 包和容器镜像等分发渠道。

## 能力范围

| 模块 | 示例能力 |
| --- | --- |
| 家庭拓扑 | 家庭、房间、区域、灯组、网关、面板、旋钮、传感器、统一实体列表 |
| 设备与范围控制 | 单设备、全屋、房间、区域、灯组的开关、亮度、色温、相对亮度/色温调节、颜色、节点属性设置和状态读取 |
| 组织管理 | 房间和设备命名、设备移动、收藏、首页排序、面板和旋钮配置 |
| 情景和自动化 | 列表、详情、执行、创建/更新/删除、启用/禁用、写后验证 |
| 产品知识 | `product.pedia.search`、说明书和 FAQ 候选资源、物模型 schema、产品定义 |
| 诊断维护 | 网关/设备诊断、升级文件、进度查询、安装和凭据诊断 |
| 本地智能 | 本地偏好记忆、推荐列表、推荐反馈、冷却和拒绝 |

只读能力会立即执行。持久化写入和删除也通过 Runtime 校验后直接执行；调用方需要先让用户确认时，可使用 `--dry-run`、`--preview-only` 或 `options.dryRun=true` 获取无写入预览。

## 安装

macOS 和 Linux:

```sh
curl -fsSL https://github.com/Yeelight/yeelight-home/releases/latest/download/install.sh | sh
```

Windows PowerShell:

```powershell
iwr https://github.com/Yeelight/yeelight-home/releases/latest/download/install.ps1 -UseB | iex
```

Homebrew:

```sh
brew install Yeelight/tap/yeelight-home
```

Scoop:

```powershell
scoop bucket add yeelight https://github.com/Yeelight/scoop-bucket
scoop install yeelight-home
```

npm wrapper:

```sh
npm install -g yeelight-home
```

国内访问 npm 较慢或失败时，可只为本次安装临时使用 npmmirror：

```sh
npm install -g yeelight-home --registry=https://registry.npmmirror.com
npm config get registry
```

如果以前修改过全局 registry，可用
`npm config set registry https://registry.npmjs.org/` 恢复官方源。npm wrapper
会使用 `checksums.txt` 校验二进制；官方仓库默认先尝试 GitHub，失败或超时后自动尝试
Gitee 和 GitCode 的官方 Release 镜像。

Debian、Ubuntu、Fedora、Arch、Winget、Docker、GHCR、Docker Hub 等渠道见 [INSTALL.md](INSTALL.md) 和 [DISTRIBUTION.md](DISTRIBUTION.md)。

## 让 AI 一句话帮你安装

如果你使用的是可以执行本地终端命令的 AI 助手，可以直接把下面这一句话发给它：

```text
请从 Yeelight 官方 GitHub Release、国内官方镜像或已支持的包管理渠道安装 `yeelight-home`。安装后运行 `yeelight-home setup --lang zh-CN`，优先选择“完整智能模式”，引导我用 Yeelight Pro APP 首页右上角 `+` -> MCP 授权扫码，并等待我完成。不要索要或打印 token、密码、Cookie、Client ID 或扫码结果；最后运行 `yeelight-home doctor --json` 和只读家庭发现验证。
```

## 快速开始

```sh
yeelight-home setup --lang zh-CN
yeelight-home setup --lang zh-CN --mode skill --agent auto --yes
yeelight-home setup --lang zh-CN --mode mcp --agent auto --yes
yeelight-home menu
yeelight-home version
yeelight-home doctor
yeelight-home doctor --json
yeelight-home auth status --json
yeelight-home auth login --qr
yeelight-home home list --json
yeelight-home home select --house-id <house-id>
yeelight-home device list --json
yeelight-home product search --multi-field 青空灯 --json
yeelight-home scene execute --scene-id <scene-id> --json
yeelight-home light on --device-id <device-id> --json
yeelight-home automation enable --automation-id <automation-id> --json
```

局域网优先模式：

```sh
yeelight-home setup --lang zh-CN --mode lan --gateway-ip 192.168.1.2 --agent auto --yes
yeelight-home lan inspect --json
yeelight-home config set --control-mode local-only --gateway-ip 192.168.1.2
```

MCP 客户端默认启动本地统一 Runtime：`yeelight-home mcp serve --stdio`。只有明确的高级兼容场景才使用 `--mcp-source cloud` 或 `--mcp-source gateway`。Cloud MCP 配置只会把本机 `yeelight-home mcp proxy` 启动参数写入 AI 客户端；代理在运行时从本地凭据存储读取 Authorization，不会把 Token 复制进客户端配置文件。

默认区域是 `cn`。如果账号属于其他区域，请加 `--region sg`、`--region us` 或 `--region eu`。

默认使用普通 Yeelight Pro 家庭（`bizType=0`）。如果你使用易来商照项目，运行 `yeelight-home setup --lang zh-CN --biz-type 1`，或在 `auth login`、`home list` 后加 `--biz-type 1`。切换家庭类型时会清空旧类型的默认家庭，并重新发现和选择当前类型的项目，不会把普通家庭 House ID 误当成商照项目 ID。

无法扫码时，可以在自己的终端中安全导入已获准的 token。真实使用时优先用 `--stdin`，避免 token 留在 shell history 中：

```sh
printf '%s' "$YEELIGHT_TOKEN" | yeelight-home auth token set --stdin --region cn
printf '%s' "$YEELIGHT_DEV_TOKEN" | yeelight-home auth token set --stdin --profile dev --region dev --json
```

不要把 token 粘贴到 AI 聊天里。CLI 会本地保存 token，`auth status`、`doctor` 和普通 Runtime 输出不会打印 token 值。

## 配置模型

配置优先级固定为：

1. 命令行 flag。
2. 环境变量。
3. 当前 profile 元数据和本地凭据存储。
4. 默认值。

默认值：

- Profile: `default`
- Region: `cn`
- 家庭类型：`0`（普通 Yeelight Pro 家庭）；商照项目使用 `1`
- Home: 未选择；只有家庭内设备、房间、情景、自动化等操作需要 `houseId`

常用环境变量：

| 变量 | 作用 |
| --- | --- |
| `YEELIGHT_HOME_BIN` | Skill 查找 CLI 时使用的绝对路径。 |
| `YEELIGHT_HOME_PROFILE` | 为当前进程选择 profile。 |
| `YEELIGHT_CLOUD_REGION` | 临时覆盖区域：`cn`、`sg`、`us`、`eu` 或开发用 `dev`。 |
| `YEELIGHT_HOME_BIZ_TYPE` | 为当前进程选择普通家庭（`0`）或商照项目（`1`）。 |
| `YEELIGHT_HOME_HOUSE_ID` | 临时覆盖默认家庭。 |
| `YEELIGHT_HOME_ACCESS_TOKEN` | 临时 token，适合本地 smoke 或 CI；不会写入 profile 元数据。 |
| `YEELIGHT_HOME_DIR` | 覆盖 Runtime home 目录。 |
| `YEELIGHT_API_BASE_URL` | 开发专用 API base URL 覆盖；不要在 Skill prompt 或用户自动化里使用。 |

`houseId` 是可选 profile 元数据。账号级命令不需要 `houseId`，例如 `auth status`、`doctor`、`api smoke`、`home list`、`home.summary`、`home.search` 和 `account.info`。家庭内操作需要从请求、`YEELIGHT_HOME_HOUSE_ID` 或已选择 profile 中得到 `houseId`。

完整配置说明见 [CONFIG.md](CONFIG.md)。

## 命令模型

面向 Skill、生成应用和自动化 host 的稳定接口是：

```sh
yeelight-home invoke --stdin [--profile <name>] [--region <region>] [--house-id <id>]
```

面向人的命令使用常规资源结构：

```text
yeelight-home <resource> <action> [--json] [--profile <name>] [--region <region>] [--house-id <id>] [resource flags]
```

常用示例：

```sh
yeelight-home device list --json
yeelight-home room list --json
yeelight-home scene execute --scene-id <scene-id> --json
yeelight-home light brightness --device-id <device-id> --brightness 60 --json
yeelight-home automation enable --automation-id <automation-id> --json
```

资源命令和 `invoke` 共用同一套校验、脱敏、直接执行、dry-run 预览和写后验证规则。

查看帮助：

```sh
yeelight-home --help
yeelight-home help device
yeelight-home help scene execute
yeelight-home help light brightness
```

查看机器可读 intent 契约：

```sh
yeelight-home intent explain --intent scene.update --json
yeelight-home intent schema --intent lighting.design.import --json
yeelight-home explain lighting.design.import --json
```

这些命令离线运行，不需要 token。它们会输出 SkillRequest 外层结构、可接受参数、复杂嵌套 payload shape、examples 和 nextStep，方便 Skill 或传统程序直接生成照明设计模型、情景 `actions[]`、自动化 `trigger` / `conditions` / `actions[]`、面板事件、批量操作等大 JSON 字段。

传递不常用参数时，不需要手写完整 SkillRequest，可以用 `--set` 或 `--params-json`：

```sh
yeelight-home room search --name 客厅 --json
yeelight-home favorite add --set targetType=device,targetId=50018330,rank=1 --json
yeelight-home product search --multi-field 青空灯 --json
yeelight-home product search --product-model YP-0117 --json
yeelight-home thing schema-get --schema-id <schema-id> --json
yeelight-home panel button-configure --device-id <panel-id> --params-json '<json>' --json
```

## 产品知识

```sh
yeelight-home product search --multi-field 青空灯 --json
yeelight-home product search --product-code 1-000003268 --json
yeelight-home product search --product-model YP-0117 --json
```

`product search` 调用产品百科搜索能力。返回结果会保留经过脱敏的产品字段，例如产品名、品牌、型号、SKU/SPU、品类/分类、产品编码、支持标记、状态、附件，以及可安全推导的说明书或 FAQ 候选资源地址。

产品知识用于解释“这个产品是什么、有什么资料、可能有什么说明书或 FAQ”。它不能证明用户家里已经安装了这个设备，也不能证明该设备当前可控。判断已安装设备能力时，应使用 `entity capabilities`、`device detail` 或 `state query` 等家庭内证据。

## 本地记忆和推荐

```sh
yeelight-home memory remember --house-id <house-id> --set scopeType=room,scopeRef=客厅,preferenceType=brightness,preferenceValue=45 --json
yeelight-home memory remember --house-id <house-id> --params-json '{"preferences":[{"scopeType":"profile","preferenceType":"ambience","preferenceValue":"prefer_romantic_warm","evidence":"用户明确要求记住喜欢浪漫色调"},{"scopeType":"profile","preferenceType":"product_preference","preferenceValue":"prefer_premium_luxury","evidence":"用户明确要求记住高端奢华产品定位"}]}' --json
yeelight-home recommendation record --house-id <house-id> --params-json '{"type":"automation","source":"ai_skill","targetIntent":"automation.create","scopeType":"room","scopeRef":"主卧","explanation":"可以把已保存的浪漫暖光偏好做成主卧晚间自动化。","evidence":"本地记忆 ambience=prefer_romantic_warm"}' --json
yeelight-home recommendation list --house-id <house-id> --json
yeelight-home recommendation feedback --house-id <house-id> --params-json '{"recommendationId":"<id>","feedback":"cooldown","cooldownHours":24}' --json
```

本地记忆和推荐默认对每个 `profile + region + houseId` 范围开启。`memory pause` 是明确的退出开关，`memory resume` 会重新开启本地学习。`memory remember` 会直接 upsert 单条结构化本地偏好，也支持通过 `parameters.preferences[]` 一次写入多条结构化偏好。`recommendation record` 会直接 upsert 调用方已经组织好的结构化推荐候选；Runtime 只负责校验、保存、去重、排序、列表和反馈记录。推荐判断属于调用方/Skill，不属于 Runtime。`accepted`、`dismissed`、`rejected` 和 `cooldown` 等反馈会被本地保存，并影响后续推荐展示。

Runtime 不会把完整对话日志当作记忆保存，也不解释主观自然语言偏好。Skill 或其他调用方必须传入 `scopeType`、`scopeRef`、`preferenceType`、`preferenceValue`、`evidence` 等结构化字段。如果调用方希望“柔和暖光”和“偏暖一点”视为同一条记忆，应先在调用方侧归一成同一个 canonical `preferenceValue`；Runtime 只负责合并完全相同的结构化偏好和证据。

记忆 JSON 会按 `~/.yeelight-home/data/memory/<profile>/<region>/<houseId>.json` 分片保存，每个分片和导出结果都会携带 `accountProfile`、`profile`、`region`、`houseId`、`dataType` 组成的 namespace 元数据。交互信号只保存 `intent` 和响应 `status` 这类粗粒度客观证据，不保存用户原话。已接受、忽略或拒绝的推荐证据和交互信号会按本地保留窗口压缩；显式偏好会保留到用户主动 forget。

默认本地数据目录是：

```text
~/.yeelight-home/data/
```

如果使用 `YEELIGHT_HOME_DIR`，记忆和 profile 数据会写入对应目录，适合测试和隔离验证。

## Skill 集成

Skill wrapper 查找顺序：

1. `YEELIGHT_HOME_BIN`
2. `PATH` 中的 `yeelight-home`

发布版 Skill 不携带 Runtime 二进制、Runtime 源码或安装脚本。用户需要先通过公开渠道安装 CLI，然后运行：

```sh
yeelight-home auth status --json
yeelight-home auth login --qr
yeelight-home home list --json
```

如果不能扫码，并且用户已经有获准 token，可在本地终端导入：

```sh
printf '%s' "$YEELIGHT_TOKEN" | yeelight-home auth token set --stdin --region cn
```

Skill 应使用 `yeelight-home` 命令，不应调用 URL、header、curl、第三方服务或带 token 的命令。

## 发布和分发

`yeelight-home` 直接维护在单体仓库顶层的 `yeelight-home/` 自包含项目中，并从同一项目发布到 `Yeelight/yeelight-home`。不再保留旧的 `yeelight-smart-home/runtime` 目录。

公开发布流程由 GoReleaser 负责。一次 `v*` tag 发布可以产出：

- macOS、Linux、Windows 的 `amd64`、`arm64`，以及 Linux `armv7` 归档。
- checksums、SBOM 和 release metadata。
- Homebrew tap Formula 和 Cask。
- Scoop bucket manifest。
- Linux `.deb`、`.rpm`、`.apk` 和 Arch 包。
- GHCR 和 Docker Hub 多架构镜像。
- 在凭据配置完成时发布或生成 Snap、AUR、Winget 相关产物。

`Yeelight/homebrew-tap` 和 `Yeelight/scoop-bucket` 是包管理器元数据仓库，不是 Runtime 源码仓库。

## 安全说明

- 不要把 token、密码或账号密钥粘贴到 AI 聊天中。
- `auth status`、`doctor` 和 `invoke` 输出会脱敏。
- profile 元数据只保存 profile 名称、region、默认 home、QR device 等非密钥信息。
- token 保存在本地凭据存储或受保护的本地 fallback 中。
- 持久化写入使用受支持的 Runtime intent；模型不能执行任意底层 payload。高影响操作的用户确认由调用方负责。
