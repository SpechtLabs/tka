import { defineClientConfig } from "vuepress/client";
import VPContributorsCustom from "./components/VPContributorsCustom.vue";
import VPListCompare from "./components/VPListCompareCustom.vue";
import VPReleasesCustom from "./components/VPReleasesCustom.vue";
import VPSwaggerUI from "./components/VPSwaggerUI.vue";

export default defineClientConfig({
  enhance({ app }) {
    app.component("VPContributors", VPContributorsCustom);
    app.component("VPReleases", VPReleasesCustom);
    app.component("VPSwaggerUI", VPSwaggerUI);
    app.component("VPListCompare", VPListCompare);
  },
});
