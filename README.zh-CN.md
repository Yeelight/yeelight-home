# yeelight-home

默认文档语言是英文：[README.md](README.md)。本文是中文使用说明。

`yeelight-home` 是 Yeelight 智能家居 Skill 和自动化脚本使用的本地 Runtime CLI。它运行在用户自己的电脑或服务器上，本地保存凭据，解析智能家居请求，执行受支持的 Yeelight 家庭能力，并返回经过脱敏的结构化结果。

Runtime 不会被打包进 Skill。Skill 只通过 `YEELIGHT_HOME_BIN` 或 `PATH` 找到公开安装的 `yeelight-home`，然后向 `yeelight-home invoke --stdin` 发送一个 JSON 请求。

## Yeelight AI 能力矩阵

这些项目组成互补的 Yeelight AI 技术栈。可以根据接入方式选择入口，也可以组合使用。

| 项目 | 定位与核心能力 | 适用场景 | GitHub |
| --- | --- | --- | --- |
| Yeelight Home | 首选本地语义 Runtime，通过统一结构化 `invoke --stdin` 边界提供查询、控制、场景、自动化、灯光设计、诊断、产品知识和生成应用能力。 | 需要稳定、受策略保护的智能家居执行层的 Agent host、本地自动化和应用。 | [Yeelight/yeelight-home](https://github.com/Yeelight/yeelight-home) |
| Yeelight Smart Home Skills | 官方 Agent Skills：Smart Home 把自然语言转换为安全的 Runtime 操作；PRO App Builder 基于已验证能力生成专用本地应用。 | 需要智能家居对话工作流或应用生成能力的 Agent host。 | [Yeelight/yeelight-smart-home-skills](https://github.com/Yeelight/yeelight-smart-home-skills) |
| Yeelight AI CLI | 统一终端工作台和 MCP 客户端，连接 Cloud、Metadata 和 LAN 服务，提供本地 profile、安全快捷命令、诊断、脚本和 AI 客户端配置。 | 希望通过通用 MCP 与自动化命令行入口操作的用户、脚本和 CI。 | [Yeelight/yeelight-cli](https://github.com/Yeelight/yeelight-cli) |
| Yeelight IoT MCP | 官方托管或可自行部署的 Streamable HTTP MCP 服务，提供拓扑、实时状态、设备控制和场景执行。 | 需要直接发现和控制 IoT 设备的 MCP 客户端。 | [Yeelight/yeelight-iot-mcp](https://github.com/Yeelight/yeelight-iot-mcp) |
| Yeelight Metadata MCP | 官方托管或可自行部署的 Streamable HTTP MCP 服务，提供受保护的家庭、房间、组、面板、场景、自动化、收藏和账号元数据工作流。 | 需要检查和管理元数据的 MCP 客户端。 | [Yeelight/yeelight-metadata-mcp](https://github.com/Yeelight/yeelight-metadata-mcp) |

Yeelight Home 还提供系统凭据存储、本地 QR 登录、秘密脱敏诊断、预览与校验、调用方确认和 Runtime 策略/写后读取、本地记忆与推荐、实操经验，以及机器可读的 intent schema 和解释。跨平台二进制通过 GitHub Release、npm 和已支持的包管理器分发。

典型组合：智能家居 Agent 和生成应用 -> Skills -> Yeelight Home；终端用户和脚本 -> Yeelight AI CLI；MCP 客户端 -> IoT MCP 和/或 Metadata MCP。

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

Debian、Ubuntu、Fedora、Arch、Winget、Docker、GHCR、Docker Hub 等渠道见 [INSTALL.md](INSTALL.md) 和 [DISTRIBUTION.md](DISTRIBUTION.md)。

## 让 AI 一句话帮你安装

如果你使用的是可以执行本地终端命令的 AI 助手，可以直接把下面这一句话发给它：

```text
请从 Yeelight 官方 GitHub Release 或已支持的包管理渠道，为我的系统安装 Yeelight Home Runtime CLI，然后从 Yeelight 官方 Skill Release 或 ClawHub 来源安装 Yeelight Smart Home Skill。安装后用 `yeelight-home doctor --json` 验证 CLI，并引导我执行 `yeelight-home auth login --qr`；不要要求我把 token、密码或 cookie 粘贴到聊天里。
```

## 快速开始

```sh
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

默认区域是 `cn`。如果账号属于其他区域，请加 `--region sg`、`--region us` 或 `--region eu`。

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
- Home: 未选择；只有家庭内设备、房间、情景、自动化等操作需要 `houseId`

常用环境变量：

| 变量 | 作用 |
| --- | --- |
| `YEELIGHT_HOME_BIN` | Skill 查找 CLI 时使用的绝对路径。 |
| `YEELIGHT_HOME_PROFILE` | 为当前进程选择 profile。 |
| `YEELIGHT_CLOUD_REGION` | 临时覆盖区域：`cn`、`sg`、`us`、`eu` 或开发用 `dev`。 |
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
