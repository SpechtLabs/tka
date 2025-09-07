import { defineClientConfig } from "vuepress/client";
import VPContributorsCustom from "./components/VPContributorsCustom.vue";
import VPReleasesCustom from "./components/VPReleasesCustom.vue";
import VPSwaggerUI from "./components/VPSwaggerUI.vue";

export default defineClientConfig({
  enhance({ app }) {
    app.component("VPContributorsCustom", VPContributorsCustom);
    app.component("VPReleasesCustom", VPReleasesCustom);
    app.component("VPSwaggerUI", VPSwaggerUI);
  },
});
