import { defineConfig } from 'vitepress'

export default defineConfig({
  base: '/yew-key/',
  lang: 'zh-CN',
  title: 'yewk',
  description: 'a cli to sync your secrets to anywhere, use system keyring to store safely',

  themeConfig: {
    nav: [
      { text: '快速开始', link: '/guide/usage' },
      { text: 'Shell 补全', link: '/guide/completion' },
      { text: '卸载', link: '/guide/uninstall' },
      { text: 'GitHub', link: 'https://github.com/YewFence/yew-key' }
    ],

    sidebar: [
      {
        text: '指南',
        items: [
          { text: '快速开始', link: '/guide/usage' },
          { text: 'Shell 补全', link: '/guide/completion' },
          { text: '卸载', link: '/guide/uninstall' }
        ]
      }
    ],

    search: {
      provider: 'local'
    },

    socialLinks: [
      { icon: 'github', link: 'https://github.com/YewFence/yew-key' }
    ],

    footer: {
      message: 'Released under the MIT License.',
      copyright: 'Copyright © YewFence'
    },

    docFooter: {
      prev: '上一页',
      next: '下一页'
    },

    outline: {
      label: '本页目录'
    },

    lastUpdated: {
      text: '最后更新'
    }
  }
})
