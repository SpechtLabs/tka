import { defineNavbarConfig } from "vuepress-theme-plume";

export const navbar = defineNavbarConfig([
  { text: "Home", link: "/", icon: "mdi:home" },

  {
    text: "Getting Started",
    icon: "mdi:rocket-launch",
    items: [
      { text: "Overview", link: "/getting-started/overview", icon: "mdi:eye" },
      { text: "Prerequisites", link: "/getting-started/prerequisites", icon: "mdi:check-circle" },
      { text: "Quick Start", link: "/getting-started/quick", icon: "mdi:flash" },
      { text: "Comprehensive Guide", link: "/getting-started/comprehensive", icon: "mdi:book-open-page-variant" },
      { text: "Troubleshooting", link: "/getting-started/troubleshooting", icon: "mdi:wrench" },
    ],
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
    items: [
      { text: "Architecture", link: "/understanding/architecture", icon: "mdi:sitemap" },
      { text: "Security Model", link: "/understanding/security", icon: "mdi:security" },
    ],
  },

  {
    text: "Reference",
    icon: "mdi:book",
    items: [
      { text: "API Reference", link: "/reference/api", icon: "mdi:api" },
      { text: "CLI Reference", link: "/reference/cli", icon: "mdi:terminal" },
      { text: "Configuration", link: "/reference/configuration", icon: "mdi:file-cog" },
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
