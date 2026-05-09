import { defineConfig } from 'vitepress'

const repo = 'https://github.com/SunBK201/UA3F'

const enSidebar = [
  {
    text: 'Guide',
    collapsed: false,
    items: [
      { text: 'Getting Started', link: '/guide/getting-started' },
      { text: 'Concepts', link: '/guide/concepts' },
      { text: 'Configuration', link: '/guide/configuration' },
      { text: 'Configuration Examples', link: '/guide/config-examples' },
      { text: 'Working with Other Proxies', link: '/guide/proxy-integration' }
    ]
  },
  {
    text: 'Server Modes',
    collapsed: false,
    items: [
      { text: 'HTTP', link: '/modes/http' },
      { text: 'SOCKS5', link: '/modes/socks5' },
      { text: 'TPROXY', link: '/modes/tproxy' },
      { text: 'REDIRECT', link: '/modes/redirect' },
      { text: 'NFQUEUE', link: '/modes/nfqueue' }
    ]
  },
  {
    text: 'HTTP Rewrite',
    collapsed: false,
    items: [
      { text: 'Rewrite Modes', link: '/http-rewrite/rewrite-modes' },
      { text: 'Match Rules', link: '/http-rewrite/match-rules' },
      { text: 'Rewrite Actions', link: '/http-rewrite/rewrite-actions' },
      { text: 'HTTPS MitM', link: '/http-rewrite/mitm' }
    ]
  },
  {
    text: 'L3 Rewrite',
    collapsed: false,
    items: [
      { text: 'Overview', link: '/l3/overview' },
      { text: 'TTL', link: '/l3/ttl' },
      { text: 'IPID', link: '/l3/ipid' },
      { text: 'TCP Timestamp', link: '/l3/tcp-timestamp' },
      { text: 'TCP Initial Window', link: '/l3/tcp-initial-window' }
    ]
  },
  {
    text: 'Desync',
    collapsed: false,
    items: [
      { text: 'Overview', link: '/desync/overview' },
      { text: 'TCP Segment Reordering', link: '/desync/tcp-reorder' },
      { text: 'TCP Obfuscation Injection', link: '/desync/tcp-inject' }
    ]
  },
  {
    text: 'eBPF Offload',
    collapsed: false,
    items: [
      { text: 'L3 Rewrite eBPF Acceleration', link: '/ebpf/l3-rewrite' }
    ]
  },
  {
    text: 'API',
    collapsed: false,
    items: [
      { text: 'API Reference', link: '/api/' }
    ]
  }
]

const zhSidebar = [
  {
    text: '指南',
    collapsed: false,
    items: [
      { text: '快速开始', link: '/zh/guide/getting-started' },
      { text: '核心概念', link: '/zh/guide/concepts' },
      { text: '配置说明', link: '/zh/guide/configuration' },
      { text: '配置示例', link: '/zh/guide/config-examples' },
      { text: '与其他代理配合', link: '/zh/guide/proxy-integration' }
    ]
  },
  {
    text: '服务模式',
    collapsed: false,
    items: [
      { text: 'HTTP', link: '/zh/modes/http' },
      { text: 'SOCKS5', link: '/zh/modes/socks5' },
      { text: 'TPROXY', link: '/zh/modes/tproxy' },
      { text: 'REDIRECT', link: '/zh/modes/redirect' },
      { text: 'NFQUEUE', link: '/zh/modes/nfqueue' }
    ]
  },
  {
    text: 'HTTP 重写',
    collapsed: false,
    items: [
      { text: '重写模式', link: '/zh/http-rewrite/rewrite-modes' },
      { text: '匹配规则', link: '/zh/http-rewrite/match-rules' },
      { text: '重写动作', link: '/zh/http-rewrite/rewrite-actions' },
      { text: 'HTTPS MitM', link: '/zh/http-rewrite/mitm' }
    ]
  },
  {
    text: 'L3 重写',
    collapsed: false,
    items: [
      { text: 'L3 重写介绍', link: '/zh/l3/overview' },
      { text: 'TTL', link: '/zh/l3/ttl' },
      { text: 'IPID', link: '/zh/l3/ipid' },
      { text: 'TCP 时间戳', link: '/zh/l3/tcp-timestamp' },
      { text: 'TCP 初始窗口', link: '/zh/l3/tcp-initial-window' }
    ]
  },
  {
    text: 'Desync',
    collapsed: false,
    items: [
      { text: 'Desync 介绍', link: '/zh/desync/overview' },
      { text: 'TCP 分片乱序发射', link: '/zh/desync/tcp-reorder' },
      { text: 'TCP 混淆注入', link: '/zh/desync/tcp-inject' }
    ]
  },
  {
    text: 'eBPF 流量卸载',
    collapsed: false,
    items: [
      { text: 'L3 重写 eBPF 加速', link: '/zh/ebpf/l3-rewrite' }
    ]
  },
  {
    text: 'API',
    collapsed: false,
    items: [
      { text: 'API 文档', link: '/zh/api/' }
    ]
  }
]

export default defineConfig({
  title: 'UA3F',
  description: 'Advanced HTTP(S) rewriting proxy',
  cleanUrls: true,
  lastUpdated: true,
  rewrites: (id) => id.startsWith('en/') ? id.slice(3) : id,
  head: [
    ['link', { rel: 'icon', href: 'https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f-210.png' }],
    ['meta', { name: 'theme-color', content: '#2563eb' }]
  ],
  markdown: {
    headers: {
      level: [2, 3]
    },
    theme: {
      light: 'github-light',
      dark: 'github-dark'
    }
  },
  themeConfig: {
    logo: 'https://sunbk201.oss-cn-beijing.aliyuncs.com/img/ua3f-210.png',
    aside: true,
    nav: [
      { text: 'English', link: '/' },
      { text: '简体中文', link: '/zh/' },
      { text: 'GitHub', link: repo }
    ],
    search: {
      provider: 'local'
    },
    socialLinks: [
      { icon: 'github', link: repo }
    ],
    footer: {
      message: 'Released under the GPL-3.0 license.',
      copyright: 'Copyright © SunBK201'
    }
  },
  locales: {
    root: {
      label: 'English',
      lang: 'en-US',
      title: 'UA3F',
      description: 'Advanced HTTP(S) rewriting proxy',
      themeConfig: {
        nav: [
          { text: 'Guide', link: '/guide/getting-started' },
          { text: 'Configuration', link: '/guide/configuration' },
          { text: 'API', link: '/api/' },
          { text: 'GitHub', link: repo }
        ],
        sidebar: {
          '/': enSidebar
        },
        outline: {
          level: [2, 3],
          label: 'Page Navigation'
        }
      }
    },
    zh: {
      label: '简体中文',
      lang: 'zh-CN',
      title: 'UA3F',
      description: '高级 HTTP(S) 重写代理',
      themeConfig: {
        nav: [
          { text: '指南', link: '/zh/guide/getting-started' },
          { text: '配置', link: '/zh/guide/configuration' },
          { text: 'API', link: '/zh/api/' },
          { text: 'GitHub', link: repo }
        ],
        sidebar: {
          '/zh/': zhSidebar
        },
        outline: {
          level: [2, 3],
          label: '页面导航'
        },
        docFooter: {
          prev: '上一页',
          next: '下一页'
        },
        lastUpdated: {
          text: '最后更新'
        },
        darkModeSwitchLabel: '主题',
        sidebarMenuLabel: '菜单',
        returnToTopLabel: '回到顶部',
        langMenuLabel: '切换语言'
      }
    }
  }
})
