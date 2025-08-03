import { viteBundler } from "@vuepress/bundler-vite";
// import registerComponentsPlugin from '@vuepress/plugin-register-components';
import { registerComponentsPlugin } from "@vuepress/plugin-register-components";
import { path } from "@vuepress/utils";
import { defineUserConfig } from "vuepress";
import { plumeTheme } from "vuepress-theme-plume";

export default defineUserConfig({
  base: "/",
  lang: "en-US",
  title: "Tailscale K8s Auth",
  description:
    "Zero-friction Kubernetes access using Tailscale and ephemeral service accounts",

  head: [
    [
      "meta",
      {
        name: "description",
        content:
          "Forget complex auth proxies, VPNs, or OIDC setups. `tailscale-k8s-auth` gives you secure, identity-aware access to your Kubernetes clusters using just your Tailscale identity and network — with short-lived, auto-cleaned credentials.",
      },
    ],
    ["link", { rel: "icon", type: "image/png", href: "/images/specht.png" }],
  ],

  bundler: viteBundler(),
  shouldPrefetch: false,

  plugins: [
    registerComponentsPlugin({
      componentsDir: path.resolve(__dirname, "./components"),
    }),
  ],

  theme: plumeTheme({
    docsRepo: "https://github.com/SpechtLabs/tailscale-k8s-auth",
    docsDir: "docs",
    docsBranch: "main",

    editLink: false,
    lastUpdated: false,
    contributors: false,

    blog: {
      postList: true,
      tags: false,
      archives: false,
      categories: false,
      postCover: "right",
      pagination: 15,
    },

    article: "/article/",

    cache: "filesystem",
    search: { provider: "local" },

    /**
     * markdown
     * @see https://theme-plume.vuejs.press/config/markdown/
     */
    markdown: {
      collapse: true,
      timeline: true,
      //   abbr: true,         // 启用 abbr 语法  *[label]: content
      //   annotation: true,   // 启用 annotation 语法  [+label]: content
      //   pdf: true,          // 启用 PDF 嵌入 @[pdf](/xxx.pdf)
      //   caniuse: true,      // 启用 caniuse 语法  @[caniuse](feature_name)
      //   plot: true,         // 启用隐秘文本语法 !!xxxx!!
      //   bilibili: true,     // 启用嵌入 bilibili视频 语法 @[bilibili](bid)
      //   youtube: true,      // 启用嵌入 youtube视频 语法 @[youtube](video_id)
      //   artPlayer: true,    // 启用嵌入 artPlayer 本地视频 语法 @[artPlayer](url)
      //   audioReader: true,  // 启用嵌入音频朗读功能 语法 @[audioReader](url)
      //   icons: true,        // 启用内置图标语法  :[icon-name]:
      //   codepen: true,      // 启用嵌入 codepen 语法 @[codepen](user/slash)
      //   replit: true,       // 启用嵌入 replit 语法 @[replit](user/repl-name)
      //   codeSandbox: true,  // 启用嵌入 codeSandbox 语法 @[codeSandbox](id)
      //   jsfiddle: true,     // 启用嵌入 jsfiddle 语法 @[jsfiddle](user/id)
      //   npmTo: true,        // 启用 npm-to 容器  ::: npm-to
      //   demo: true,         // 启用 demo 容器  ::: demo
      repl: {
        // 启用 代码演示容器
        go: true, // ::: go-repl
        rust: true, // ::: rust-repl
        //     kotlin: true,     // ::: kotlin-repl
      },
      //   math: {             // 启用数学公式
      //     type: 'katex',
      //   },
      //   chartjs: true,      // 启用 chart.js
      //   echarts: true,      // 启用 ECharts
      mermaid: true, // 启用 mermaid
      //   flowchart: true,    // 启用 flowchart
      image: {
        figure: true, // 启用 figure
        lazyload: true, // 启用图片懒加载
        mark: true, // 启用图片标记
        size: true, // 启用图片大小
      },
      //   include: true,      // 在 Markdown 文件中导入其他 markdown 文件内容
      //   imageSize: 'local', // 启用 自动填充 图片宽高属性，避免页面抖动
    },

    watermark: false,
  }),
});
