import { defineNavbarConfig } from "vuepress-theme-plume";

export const navbar = defineNavbarConfig([
  { text: "Home", link: "/", icon: "mdi:home" },

  {
    text: "Getting Started",
    icon: "mdi:rocket-launch",
    link: "/getting-started/overview",
  },

  {
    text: "Guides",
    icon: "mdi:compass",
    items: [
      { text: "Configure ACLs", link: "/guides/configure-acl", icon: "mdi:shield-lock" },
      { text: "Shell Integration", link: "/guides/shell-integration", icon: "mdi:console" },
      { text: "Use Subshell", link: "/guides/use-subshell", icon: "mdi:layers" },
      { text: "CLI Autocompletion", link: "/guides/autocompletion", icon: "mdi:keyboard" },
      { text: "Configure Settings", link: "/guides/configure-settings", icon: "mdi:cog" },
      { text: "Troubleshooting", link: "/guides/troubleshooting", icon: "mdi:bug" },
    ],
  },

  {
    text: "Understanding",
    icon: "mdi:lightbulb",
    link: "/understanding/architecture",
  },

  {
    text: "Reference",
    icon: "mdi:book",
    items: [
      { text: "Configuration", link: "/reference/configuration", icon: "mdi:file-cog" },
      { text: "API Reference", link: "/reference/api", icon: "mdi:api" },
      { text: "CLI Reference", link: "/reference/cli", icon: "mdi:terminal" },
      { text: "Developer Docs", link: "/reference/developer/architecture", icon: "mdi:code-braces" },
    ],
  },

  {
    text: "More",
    icon: "mdi:dots-horizontal",
    items: [
      {
        text: "Download",
        link: "https://github.com/spechtlabs/tka/releases",
        target: "_blank",
        rel: "noopener noreferrer",
        icon: "mdi:download",
      },
      {
        text: "Report an Issue",
        link: "https://github.com/spechtlabs/tka/issues/new/choose",
        target: "_blank",
        rel: "noopener noreferrer",
        icon: "mdi:bug-outline",
      }
    ],
  },
]);
