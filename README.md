# yew-key

[![Release](https://img.shields.io/github/v/release/YewFence/yew-key?sort=semver)](https://github.com/YewFence/yew-key/releases)
[![Docs](https://img.shields.io/badge/docs-online-blue)](https://YewFence.github.io/yew-key/)
[![License](https://img.shields.io/github/license/YewFence/yew-key)](LICENSE)

a CLI to sync secrets from trusted managers into your local system keyring and load them into terminal sessions without writing secret values to plaintext files

> [!NOTE]
> 本项目目前处于早期开发阶段，核心功能可能缺失，无法保证向后兼容性。

## 快速开始

### 安装

#### Mise

```bash
mise use -g github:YewFence/yew-key
```

#### Go

```bash
go install github.com/YewFence/yew-key/cmd/yewk@latest
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

1. 交互式创建一个 profile，配置 secrets 的 provider 与 Provider 中的各个参数，例如ProjectId、环境名称、路径和变量映射等。

> 不会存储认证凭证

```bash
yewk profile add
```

2. 同步远端 secret 到本机系统 keyring。

> 支持的 Secrets Manager：
> - [Infisical](https://infisical.com)
> - [OpenBao](https://www.openbao.org)

Infisical 认证读取当前进程里的 `INFISICAL_TOKEN`，OpenBao 认证读取 `BAO_TOKEN` 或 `VAULT_TOKEN`。

```bash
# 以 Infisical 为例
infisical login
export INFISICAL_TOKEN=$(infisical user get token --plain)
yewk sync work
```

3. 查看当前 profile 会加载哪些环境变量

默认不会把 secret value 打印到终端。

```bash
yewk env work --shell zsh
```

4. 配置 Secrets 自动加载

将如下命令添加到 shell 配置文件中（如 `~/.zshrc`），每次打开新 shell 时会自动从 system keyring 加载环境变量，无需联网/认证。

```bash
# 以 zsh 为例
eval $(yewk env work --shell zsh --reveal)
```

## 配置示例

配置文件默认放在 XDG 配置目录的 `yewk/config.toml`。

### Infisical profile 示例。

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

### OpenBao KV v2 profile 示例。

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

### 高级 keyring 配置

默认情况下，yewk 不会指定 Linux Secret Service 的 collection 名称，而是让系统 keyring 按当前桌面环境选择默认位置。需要固定 collection 时，可以手动编辑配置文件并增加 `keyring_collection`。

```toml
[[profiles]]
name = "work"
provider = "infisical"
keyring_service = "yewk"
keyring_collection = "kdewallet"
```

`keyring_collection` 只影响 Linux Secret Service 后端的 collection 选择，交互式 `profile add` 不会询问这个字段。

## 安全边界

1. yewk 只把业务 secret 的本地副本写入系统 keyring，不保存 Infisical token、OpenBao token、machine identity client secret 或 AppRole secret id。
2. `env` 默认只输出变量名，`--reveal` 输出的内容会进入当前 shell 环境，并按环境变量机制被子进程继承。
3. 只在显式执行 `sync` 命令时才会连接网络从远端 secret manager 获取 secret 并写入系统 keyring，`env` 和 `status` 等命令不会联网。

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

yewk 只把业务 secret 的本地副本写入系统 keyring，不保存 Infisical token、OpenBao token、machine identity client secret 或 AppRole secret id。`env` 默认只输出变量名，`--reveal` 输出的内容会进入当前 shell 环境，并按环境变量机制被子进程继承。

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
