import { defineNavbarConfig } from "vuepress-theme-plume";

export const navbar = defineNavbarConfig([
  { text: "Home", link: "/" },

  {
    text: "Overview",
    items: [
      { text: "Overview", link: "/explanation/overview" },
      { text: "Architecture", link: "/explanation/architecture" },
      { text: "Security Model", link: "/explanation/security" },
    ],
  },

  {
    text: "Tutorials",
    items: [
      {
        text: "Getting Started with TKA",
        items: [
          { text: "Quick Start", link: "/tutorials/quick" },
          { text: "Comprehensive Guide", link: "/tutorials/comprehensive" },
        ],
      },
    ],
  },

  {
    text: "How-to Guides",
    items: [
      { text: "Configure ACLs", link: "/how-to/configure-acl" },
      { text: "Shell Integration", link: "/how-to/shell-integration" },
      { text: "CLI Autocompletion", link: "/how-to/autocompletion" },
      { text: "Production Deployment", link: "/how-to/deploy-production" },
      { text: "Configure Settings", link: "/how-to/configure-settings" },
      { text: "Troubleshooting", link: "/how-to/troubleshooting" },
    ],
  },

 {
    text: "Reference",
    items: [
      {
        text: "API & CLI", items: [
          { text: "API Reference", link: "/reference/api" },
          { text: "CLI Reference", link: "/reference/cli" },
          { text: "Configuration", link: "/reference/configuration" },
        ]
      },
        {
          text: "Developer", items: [
            { text: "Architecture", link: "/reference/developer/architecture" },
            { text: "Shell Integration Details", link: "/reference/developer/shell-integration" },
            { text: "Request Flows", link: "/reference/developer/request-flows" },
            { text: "pkg/lnhttp", link: "/reference/developer/lnhttp-server" },
            { text: "pkg/tailscale", link: "/reference/developer/tailscale-server" },
          ]
        },
    ],
  },

  {
    text: "More",
    items: [
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
      }
    ],
  },
]);
