---
layout: home

hero:
  name: 'yewk'
  text: '把远端 secrets 同步到本机系统 keyring'
  tagline: '从 Infisical 或 OpenBao 拉取 secret，本机只保存受 keyring 保护的副本，shell 启动时快速加载。'
  actions:
    - theme: brand
      text: 快速开始
      link: /guide/usage
    - theme: alt
      text: Shell 补全
      link: /guide/completion
    - theme: alt
      text: 卸载
      link: /guide/uninstall

features:
  - title: 本地缓存
    details: sync 从远端读取 secret 并写入系统 keyring，远端仍然是可信源。
  - title: 默认隐藏
    details: env 默认只显示变量名，只有传入 --reveal 才输出可执行的 export 脚本。
  - title: 明确映射
    details: profile 通过 TOML 描述远端 key 到环境变量名的映射，避免把全部远端 key 灌进 shell。
---
