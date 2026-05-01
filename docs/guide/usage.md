# 快速开始

yewk 面向开发者工作站使用，远端 secret manager 继续作为可信源，本机只保存同步后的业务 secret 副本。

## 创建 Profile

使用交互式向导创建 profile。

```bash
yewk profile add
```

profile 会写入 XDG 配置目录下的 `yewk/config.toml`，里面只保存 provider、项目、环境、路径、keyring service name 和变量映射这类非敏感信息。

也可以用参数快速创建 Infisical profile。

```bash
yewk profile add \
  --name work \
  --provider infisical \
  --infisical-project-id ... \
  --infisical-environment dev \
  --infisical-secret-path / \
  --env DATABASE_URL=DATABASE_URL \
  --env OPENAI_API_KEY=OPENAI_API_KEY
```

OpenBao KV v2 profile 可以这样创建。

```bash
yewk profile add \
  --name bao-dev \
  --provider openbao \
  --openbao-address https://bao.example.com \
  --openbao-mount secret \
  --openbao-path apps/api \
  --openbao-kv-version 2 \
  --env DATABASE_URL=DATABASE_URL
```

需要直接修改配置时使用编辑命令。

```bash
yewk profile edit
```

## 准备认证

yewk 不实现远端登录流程，也不会保存远端认证材料。Infisical 使用当前进程里的 `INFISICAL_TOKEN`。

```bash
export INFISICAL_TOKEN="$(infisical login --method=universal-auth --client-id ... --client-secret ... --silent --plain)"
```

OpenBao 使用当前进程里的 `BAO_TOKEN` 或 `VAULT_TOKEN`，地址优先读取 profile 里的 `address`，没有配置时再读取 `BAO_ADDR` 或 `VAULT_ADDR`。

```bash
export BAO_TOKEN="..."
```

## 同步到 Keyring

同步会从远端读取 profile 中映射到的 secret，并写入本机系统 keyring。

```bash
yewk sync work
```

同步成功后会更新 XDG state 目录下的 `yewk/state.json`，状态文件只保存上次成功时间、最近错误、ETag 或版本、已同步变量名等非敏感元数据。

## 输出 Shell 加载脚本

默认模式只显示会加载哪些变量名，不输出 secret value。

```bash
yewk env work --shell zsh
```

输出类似下面这样。

```zsh
# yewk profile work has 2 variables:
# DATABASE_URL
# OPENAI_API_KEY

# To load secret values in zsh, add the following line to your shell startup file:
# eval "$(yewk env work --shell zsh --reveal)"
```

确认需要加载 secret value 时，显式传入 `--reveal`。

```bash
eval "$(yewk env work --shell zsh --reveal)"
```

当前阶段支持 `zsh` 和 `bash`。

## 查看状态

```bash
yewk status work
```

如果同步失败，yewk 不会清理旧 secret，本地 keyring 里上一次可用的副本仍然保留。

## 配置示例

Infisical profile 示例。

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

OpenBao profile 示例。

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

## 安全边界

yewk 不是 Infisical CLI 或 OpenBao CLI 的替代品，它只使用用户已经准备好的认证上下文读取用户有权限访问的 secret。`.zshrc` 里不应该出现明文 secret，只应该加入 `eval "$(yewk env work --shell zsh --reveal)"` 这样的加载命令。
