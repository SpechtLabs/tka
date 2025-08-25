import { defineNavbarConfig } from "vuepress-theme-plume";

export const navbar = defineNavbarConfig([
  { text: "Home", link: "/" },

  {
    text: "Getting Started",
    link: "/guide/getting-started",
  },

  {
    text: "Architecture",
    items: [
      { text: "Overview", link: "/architecture/overview" },
      { text: "Security", link: "/architecture/security" },
      { text: "Authentication Model", link: "/architecture/authentication-model" },
    ],
  },

  {
    text: 'Reference',
    items: [
      { text: 'CLI Reference', link: '/reference/cli' },
      { text: 'Server Configuration', link: '/reference/configuration' },
    ],
  },

  {
    text: "Download",
    link: "https://github.com/spechtlabs/tka/releases",
    target: "_blank",
    rel: "noopener noreferrer",
  },

  {
    text: "Report an Issue",
    link: "https://github.com/spechtlabs/tka/issues/new/choose",
    target: "_blank",
    rel: "noopener noreferrer",
  },
]);
