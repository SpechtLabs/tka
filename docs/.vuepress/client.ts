import { defineClientConfig } from "vuepress/client";

export default defineClientConfig({
  enhance() {
    // The shared components (Terminal, ListCompare, SwaggerUI and the
    // Releases and Contributors sections) come from @spechtlabs/docs-kit,
    // which registers them itself (see config.ts)
  },
});
