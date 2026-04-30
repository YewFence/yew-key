# yewk

[![Release](https://img.shields.io/github/v/release/YewFence/yew-key?sort=semver)](https://github.com/YewFence/yew-key/releases)
[![Docs](https://img.shields.io/badge/docs-online-blue)](https://YewFence.github.io/yew-key/)
[![License](https://img.shields.io/github/license/YewFence/yew-key)](LICENSE)

a cli to sync your secrets to anywhere, use system keyring to store safely

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

```bash
yewk
yewk version
```

生成 Shell 补全脚本。

```bash
yewk completion zsh > _yewk
yewk completion bash > yewk.bash
yewk completion fish > yewk.fish
yewk completion powershell > yewk.ps1
```

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
