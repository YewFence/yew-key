# yewk

[![Release](https://img.shields.io/github/v/release/YewFence/yew-key?sort=semver)](https://github.com/YewFence/yew-key/releases)
[![Docs](https://img.shields.io/badge/docs-online-blue)](https://YewFence.github.io/yew-key/)
[![License](https://img.shields.io/github/license/YewFence/yew-key)](LICENSE)

a cli to sync your secrets from trusted secret managers into your local system keyring

> [!NOTE]
> 本项目目前处于早期开发阶段，核心功能可能缺失，无法保证向后兼容性。

## 快速开始

### 安装

#### Mise

```bash
# 仅在当前目录生效，如果需要安装到全局，请加上 -g 参数
mise use github:YewFence/yew-key
```

#### 从源码构建

```bash
git clone https://github.com/YewFence/yew-key.git
cd yew-key
mise trust
mise install
mise run build
```

### 使用

创建一个 profile，profile 只保存 provider、项目、环境、路径和变量映射这类非敏感配置。

```bash
yewk profile add
```

同步远端 secret 到本机系统 keyring。Infisical 认证读取当前进程里的 `INFISICAL_TOKEN`，OpenBao 认证读取 `BAO_TOKEN` 或 `VAULT_TOKEN`，yewk 不会保存这些认证材料。

```bash
export INFISICAL_TOKEN="$(infisical login --method=universal-auth --client-id ... --client-secret ... --silent --plain)"
yewk sync work
```

查看当前 profile 会加载哪些环境变量，默认不会把 secret value 打到终端。

```bash
yewk env work --shell zsh
```

明确需要加载到 shell 时再使用 `--reveal`。

```bash
eval "$(yewk env work --shell zsh --reveal)"
```

查看同步状态。

```bash
yewk status work
```

生成 Shell 补全脚本可参考[Shell 补全文档](docs/guide/completion.md)。

## 配置示例

配置文件默认放在 XDG 配置目录的 `yewk/config.toml`。下面是 Infisical profile 示例。

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

下面是 OpenBao KV v2 profile 示例。

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

yewk 只把业务 secret 的本地副本写入系统 keyring，不保存 Infisical token、OpenBao token、machine identity client secret 或 AppRole secret id。`env` 默认只输出变量名，`--reveal` 输出的内容会进入当前 shell 环境，并按环境变量机制被子进程继承。

## 文档

更多信息可查阅[文档站](https://YewFence.github.io/yew-key)

## 开发

### 依赖

推荐使用 [mise](https://github.com/jdx/mise) 管理开发工具。

本项目需要的开发工具由 [mise.toml](mise.toml) 声明，执行 `mise install` 即可安装到当前项目环境。不使用 mise 时，请参考 `mise.toml` 中的工具链接和版本自行安装。

### 常用命令

完整的 task 列表可运行 `mise tasks` 查看。

#### 主程序

```bash
# 运行命令行程序
mise run run
# 整理 Go 模块依赖
mise run tidy
# 运行格式检查、静态检查、构建和 lint
mise run check
# 本地构建可执行文件，构建产物会输出到 `bin/` 目录
mise run build
```

#### 文档站

```bash
# 安装依赖
mise run docs:install
# 本地启动文档站开发服务器
mise run docs:dev
```

#### GitHub Actions 维护

Github Action 会由[该工作流](.github/workflows/action-update.yml) 在本仓库 PR 打开时自动更新。也可以通过以下命令交互式更新 GitHub Action 版本。

> 从其他仓库打开 PR 时会只会检测 Action 版本，不会自动更新。

```bash
mise run action:update
```

#### 发布

推送到 `main` 后，Release 工作流会根据 Conventional Commits 解析版本。当存在需要发布的变更时，会自动创建 `v*` 标签、构建多平台二进制文件并发布到 GitHub Release。

```bash
git push origin main
```

也可以推送指定的 `v*` 标签，或在 GitHub Actions 页面手动触发 Release 工作流并输入要发布的标签。

```bash
git tag v0.1.0
git push origin v0.1.0
```

## 许可证

[MIT License](LICENSE)
