import { defineNavbarConfig } from "vuepress-theme-plume";

export const navbar = defineNavbarConfig([
  { text: "Home", link: "/" },

  {
    text: "Getting Started",
    items: [
      { text: "Overview", link: "/guide/overview" },
      //     { text: 'Quick Start', link: '/guide/quickstart' },
      //     { text: 'CLI & Server Usage', link: '/guide/usage' },
    ],
  },
  //
  // {
  //   text: 'Configuration',
  //   items: [
  //     { text: 'Server', link: '/config/server' },
  //     { text: 'Calendars', link: '/config/calendars' },
  //     { text: 'Rules Engine', link: '/config/rules' },
  //     { text: 'Home Assistant Add-On', link: '/config/home_assistant' },
  //   ],
  // },

  {
    text: "Download",
    link: "https://github.com/SpechtLabs/tailscale-k8s-auth/releases",
    target: "_blank",
    rel: "noopener noreferrer",
  },

  {
    text: "Report an Issue",
    link: "https://github.com/SpechtLabs/tailscale-k8s-auth/issues/new/choose",
    target: "_blank",
    rel: "noopener noreferrer",
  },
]);
