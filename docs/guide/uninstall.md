# 卸载

卸载 yewk 时建议先清理本机缓存的 secret，再移除 shell 启动项、配置状态文件和二进制文件。远端 secret manager 里的数据不会被 yewk 删除。

## 清理缓存

清理所有 profile 已经同步到系统 keyring 的本地 secret 副本。

```bash
yewk clean --all
```

也可以只清理单个 profile。

```bash
yewk clean work
```

`clean` 只删除 yewk 写入系统 keyring 的本地缓存，并清理本地同步状态，不会删除 `config.toml`，也不会连接 Infisical 或 OpenBao。

## 移除 Shell 加载

如果曾经把 `env` 加载命令加入到 `~/.zshrc`、`~/.bashrc` 或其他 shell 启动文件中，需要删除对应行。

```sh
eval "$(yewk env work --shell zsh --reveal)"
```

如果安装过 shell 补全，也可以删除自己写入的补全脚本。

```bash
rm -f ~/.zsh/completions/_yewk
rm -f ~/.bash_completion.d/yewk.bash
rm -f ~/.config/fish/completions/yewk.fish
```

## 删除配置和状态

配置文件默认位于 XDG 配置目录下的 `yewk/config.toml`，可以先用命令确认路径。

```bash
yewk profile
```

确认不再需要本机 profile 配置后，删除配置目录和状态目录。

```bash
rm -rf "${XDG_CONFIG_HOME:-$HOME/.config}/yewk"
rm -rf "${XDG_STATE_HOME:-$HOME/.local/state}/yewk"
```

这些文件只保存 profile 配置和同步元数据，不保存远端认证材料。业务 secret 的本地副本应该先通过 `yewk clean --all` 从系统 keyring 中删除。

## 移除二进制

如果通过 mise 安装到当前项目配置，可以从当前目录的 mise 配置里移除。

```bash
mise unuse github:YewFence/yew-key
```

如果通过 mise 全局安装，可以从全局配置里移除。

```bash
mise unuse -g github:YewFence/yew-key
```

如果通过 Go 安装，可以删除 Go bin 目录里的可执行文件。

```bash
GOBIN="$(go env GOBIN)"
if [ -z "$GOBIN" ]; then
  GOBIN="$(go env GOPATH)/bin"
fi
rm -f "$GOBIN/yewk"
```
