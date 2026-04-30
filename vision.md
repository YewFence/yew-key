# yewk Vision

## 1. 目标

yewk 要做成一个面向开发者工作站的 secrets 同步工具。远端系统继续作为可信源，当前优先支持 OpenBao 和 Infisical，本机只保存经过系统密钥环保护的副本，shell 启动时只需要从本机 keyring 快速加载需要的敏感环境变量。

README 开头的定位可以落成两件事。第一是 `sync` 从远端读取 secrets 并写入本地 keyring，第二是 `env` 从 keyring 输出 shell 可执行的加载脚本，并在脚本末尾用注释告诉用户如何把加载命令加入 `.zshrc`，而不是提供单独修改 shell 配置文件的命令。

## 2. 用户体验

开发者先通过交互式向导配置一个 profile，profile 描述远端 provider、项目、环境、路径和要同步成环境变量的映射关系。yewk 不需要为 profile 做完整增删改查，第一阶段只提供 `profile add` 和 `profile edit`。`profile add` 负责引导用户创建新 profile，`profile edit` 负责打开配置文件，让用户直接修改 TOML。

```bash
yewk profile add
export INFISICAL_TOKEN="$(infisical login --method=universal-auth --client-id ... --client-secret ... --silent --plain)"
yewk sync <profile>
yewk env <profile> --shell zsh
yewk env <profile> --shell zsh --reveal
yewk profile edit
```

`<profile>` 是用户自己命名的配置档，例如 `work`、`personal` 或 `acme-prod`。`yewk env <profile> --shell zsh` 默认不输出 secret value，只展示会加载哪些变量名，并在末尾给出接入提示。用户明确加上 `--reveal` 时才输出可以执行的 `export` 脚本，这样不会因为随手运行命令就把 secret 打到终端。

```zsh
# yewk profile <profile> has 2 variables:
# DATABASE_URL
# OPENAI_API_KEY

# To load secret values in zsh, add the following line to ~/.zshrc:
# eval "$(yewk env <profile> --shell zsh --reveal)"
```

`yewk env <profile> --shell zsh --reveal` 从系统 keyring 读取本地副本，输出 `export NAME='value'` 和接入提示注释。这个命令不访问网络，所以新开终端时速度应该接近只读本地 keyring 的耗时。

## 3. 配置模型

配置文件放在 XDG 目录下，建议使用 `$XDG_CONFIG_HOME/yewk/config.toml`，没有 XDG 环境变量时按平台约定回退。配置里只保存非敏感信息，例如 provider 类型、服务地址、项目编号、环境名、secret path、同步范围、变量名映射、keyring 后端偏好和 profile 名称。

远端认证不写入 yewk 配置，也不写入 yewk keyring。Infisical token、OpenBao token、machine identity client secret、AppRole secret id 这些身份材料由用户自己通过官方 CLI、环境变量或外部命令准备，yewk 只在当前进程里读取它们。同步下来的业务 secret 值必须放进 keyring。状态文件可以放在 `$XDG_STATE_HOME/yewk/state.json`，只保存非敏感同步元数据，例如上次同步时间、ETag、远端版本号和已同步变量清单。

配置建议从一开始就支持显式映射，避免把远端所有 key 无脑灌进 shell。

`profile add` 默认进入交互式流程，逐步询问 profile 名称、provider、远端定位信息、keyring service name 和环境变量映射。命令行 flag 可以作为快捷输入，但不是主要体验。`profile edit` 使用 `$VISUAL` 或 `$EDITOR` 打开配置文件，两个环境变量都没有时再使用平台默认编辑器或给出配置文件路径。

```toml
[[profiles]]
name = "work"
provider = "infisical"
keyring_service = "yewk"

[profiles.infisical]
site_url = "https://app.infisical.com"
project_id = "..."
environment = "dev"
secret_path = "/"
recursive = true
include_imports = true

[[profiles.env]]
remote_key = "DATABASE_URL"
env_name = "DATABASE_URL"

[[profiles.env]]
remote_key = "OPENAI_API_KEY"
env_name = "OPENAI_API_KEY"
```

OpenBao 的 profile 也只保存同步定位信息，认证仍然来自当前环境里的 `BAO_TOKEN`、`VAULT_TOKEN` 或后续的 `credential_command`。

```toml
[[profiles]]
name = "bao-dev"
provider = "openbao"
keyring_service = "yewk"

[profiles.openbao]
address = "https://bao.example.com"
mount = "secret"
path = "apps/api"
kv_version = 2

[[profiles.env]]
remote_key = "DATABASE_URL"
env_name = "DATABASE_URL"
```

## 4. 远端 Provider

Provider 层定义统一接口，`Fetch` 返回规范化后的 secret 列表，CLI 不直接依赖某个远端 SDK 的类型。

```go
type Provider interface {
    Fetch(ctx context.Context, profile Profile) ([]Secret, SyncCursor, error)
}

type Secret struct {
    RemoteKey string
    EnvName   string
    Value     string
    Version   string
    Source    string
}
```

Infisical 使用官方 Go SDK `github.com/infisical/go-sdk`。yewk 不实现 Infisical 登录流程，只读取当前进程里的 `INFISICAL_TOKEN` 并调用 `Auth().SetAccessToken`，再通过 `Secrets().ListSecrets` 或 `Secrets().Retrieve` 按 profile 里的 `project_id`、`environment` 和 `secret_path` 拉取数据。需要 Universal Auth、OIDC、Kubernetes 或 AWS IAM 时，用户先用 Infisical CLI 或自己的脚本获得 token。

OpenBao 使用官方 Go API `github.com/openbao/openbao/api/v2`。yewk 不实现 OpenBao 登录流程，只从当前环境读取 `BAO_TOKEN` 或 `VAULT_TOKEN`，地址优先使用 profile 里的 `address`，没有配置时再读 `BAO_ADDR` 或 `VAULT_ADDR`。KV v2 读取优先走 `client.KVv2(mount).Get(ctx, path)`，KV v1 后续按 `kv_version` 补齐。

如果只靠环境变量不够，可以在后续加入 `credential_command`，但它的边界必须很窄。命令只负责向 stdout 返回短期 token，yewk 只在当前 `sync` 进程内使用，不缓存命令输出，不接管 token 刷新。

## 5. 本地 Keyring

keyring 层建议优先采用 `github.com/99designs/keyring`，它提供统一接口和 `Keys` 能力，并支持 macOS Keychain、Windows Credential Manager、Linux Secret Service、KWallet、Pass、加密文件和 KeyCtl。相比 `github.com/zalando/go-keyring`，它更适合这个项目后续同时支持桌面系统和无桌面 Linux。

keyring 的 service name 固定默认为 `yewk`，item key 使用稳定命名空间。

```text
profiles/<profile>/env/<env_name>
profiles/<profile>/meta/index
```

`meta/index` 只保存变量名、远端 key、provider、版本和更新时间，不保存 secret value。这样 `env` 命令可以先读 index，再逐项读取真正的 secret。删除同步项时，`sync --prune` 可以根据 index 移除远端已经不存在或配置里已经删掉的本地条目。

Linux 上 Secret Service 依赖桌面会话和 D-Bus，服务器场景可能没有可用 keyring。这个工具应该允许用户选择 `pass` 或 encrypted file 后端，但默认仍优先系统原生后端，并在没有可用后端时给出明确错误。

## 6. Shell 加载

`env` 命令默认输出当前 profile 将加载的变量名和接入提示，不输出 secret value。第一阶段支持 zsh 和 bash，fish 与 PowerShell 后续增加。用户显式传入 `--reveal` 时才输出当前 shell 可执行的加载脚本，脚本输出必须使用成熟 shell 转义库处理值，建议使用 `mvdan.cc/sh/v3/shell`，不要手写引号拼接。

`env` 不负责修改 `.zshrc`。它只在输出末尾给出注释，告诉用户可以把 `eval "$(yewk env <profile> --shell zsh --reveal)"` 加入 `.zshrc`。这样 CLI 不需要碰用户的 shell 配置文件，也不会把 secret 明文落到配置文件里。

`env --reveal` 输出 export 语句和注释提示，错误信息写 stderr。默认模式只显示将加载哪些变量名和 `.zshrc` 接入提示，不显示值。

## 7. 同步流程

`sync` 读取 profile 后，从当前进程环境或 `credential_command` 获取远端认证上下文，再访问远端 provider 拉取 secrets，然后按映射规则过滤和重命名，最后写入 keyring，并更新非敏感 index。写入过程应该尽量做到单 profile 内串行，避免多个终端同时同步时互相覆盖。

当远端支持缓存标识时要复用。Infisical Go SDK 的 `ListSecrets` 返回 ETag，可以记录在 state 里，后续实现条件请求或跳过无变化写入。OpenBao KV v2 返回版本元数据，可以记录版本号，避免每次都写 keyring。

失败时不要清理旧 secret。同步失败意味着本地缓存保持上一次可用状态，`env` 仍能加载旧值，但 `yewk status <profile>` 应该能显示上次成功同步时间和最近错误。

## 8. 推荐依赖

| 用途 | 推荐库 | 原因 |
| --- | --- | --- |
| CLI | `github.com/spf13/cobra` | 项目已使用，继续复用 |
| Infisical | `github.com/infisical/go-sdk` | 官方 Go SDK，支持 access token 和 secrets API |
| OpenBao | `github.com/openbao/openbao/api/v2` | 官方 Go API，支持 KV v1 和 KV v2 |
| Keyring | `github.com/99designs/keyring` | 后端覆盖更广，接口包含 `Keys` |
| 配置路径 | `github.com/adrg/xdg` | 按平台处理 XDG 和回退目录 |
| TOML | `github.com/pelletier/go-toml/v2` | 轻量成熟，适合直接映射结构体 |
| Shell 转义 | `mvdan.cc/sh/v3/shell` | 避免手写 shell quoting |

## 9. 第一阶段范围

第一阶段实现最小可用闭环。支持交互式 `profile add`、`profile edit`、`sync`、`env` 和 `status`。Provider 支持 Infisical access token 和 OpenBao token auth，但 token 都来自用户已经准备好的环境变量。Keyring 使用 `99designs/keyring` 自动选择默认后端。配置使用 TOML。测试覆盖 provider mock、keyring mock、shell 输出转义、交互式 profile 创建和 `.zshrc` 接入提示输出。

第二阶段补齐体验。加入 `credential_command`、include imports、recursive、sync prune、fish 与 PowerShell 输出。profile 修改继续优先通过 `profile edit` 完成，不急着补完整增删改查。再往后才考虑动态 secret lease、后台自动同步和文件模板渲染。OpenBao AppRole 和 Infisical Universal Auth 不应该变成 yewk 的内置登录命令，除非未来有明确理由证明只读 token 上下文不能满足使用场景。

## 10. 安全边界

yewk 不是远端 secret manager 的替代品，也不是 Infisical CLI 或 OpenBao CLI 的替代品。它只是用用户已经准备好的认证上下文，把开发者已经有权限读取的 secrets 缓存在本机系统 keyring。`.zshrc` 里永远不能出现明文 secret。`env` 输出的 secret 会进入当前 shell 环境，因此它会被子进程继承，这个行为符合环境变量工作方式，但不能宣称比环境变量本身更安全。

日志默认不能打印 secret value。错误消息里只出现 profile、provider、远端 key 和 env name。任何 `--verbose` 输出也不能包含 value。测试中使用 mock keyring 或临时 file backend，不能依赖开发者真实 keychain。

## 11. 资料依据

Infisical 官方 Go SDK 文档和仓库说明了 `github.com/infisical/go-sdk` 是官方 Go SDK，并提供 Universal Auth 与 secrets 读取能力，见 https://infisical.com/docs/sdks/languages/go 和 https://github.com/infisical/go-sdk。

OpenBao 官方仓库的 API 包说明了 `github.com/openbao/openbao/api` 用于和 OpenBao 或 Vault 服务交互，OpenBao 文档示例使用 `github.com/openbao/openbao/api/v2` 和 KV v2 读取，见 https://github.com/openbao/openbao/tree/main/api 和 https://openbao.org/docs。

`99designs/keyring` 仓库说明它支持 macOS Keychain、Windows Credential Manager、Secret Service、KWallet、Pass、encrypted file 和 KeyCtl，并提供统一的 `Set`、`Get`、`Keys` 接口，见 https://github.com/99designs/keyring。

`zalando/go-keyring` 仓库说明它支持 macOS、Linux 或 BSD 的 Secret Service D-Bus，以及 Windows，接口更简单但后端较少，见 https://github.com/zalando/go-keyring。
