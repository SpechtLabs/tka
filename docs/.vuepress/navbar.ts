import { defineNavbarConfig } from "vuepress-theme-plume";

export const navbar = defineNavbarConfig([
  { text: "Home", link: "/" },

  {
    text: "Overview",
    items: [
      { text: "Overview", link: "/overview/overview" },
      { text: "Security", link: "/overview/security" },
    ],
  },

  {
    text: "Guides",
    items: [
      { text: "Getting Started", link: "/guide/getting-started" },
    ],
  },

  {
    text: 'Reference',
    items: [
      {
        text: 'Application Documentation', items: [
          { text: 'CLI Reference', link: '/reference/cli' },
          { text: 'Server Configuration', link: '/reference/configuration' },
        ]
      },
      {
        text: 'Developer Documentation', items: [
          { text: 'Request Flows', link: '/reference/developer/request-flows' },
        ]
      },
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
