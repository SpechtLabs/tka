import { defineThemeConfig } from "vuepress-theme-plume";
import { navbar } from "./navbar";

/**
 * @see https://theme-plume.vuejs.press/config/basic/
 */
export default defineThemeConfig({
  logo: "/images/specht.png",

  appearance: true, // Configure Dark Mode

  social: [
    {
      icon: "github",
      link: "https://github.com/spechtlabs/tka",
    },
  ],
  navbarSocialInclude: ["github"],
  aside: true,

  prevPage: true,
  nextPage: true,
  createTime: true,

  footer: {
    message:
      '<a target="_self" href="https://specht-labs.de/impressum/">Impressum</a> - <a target="_self" href="https://specht-labs.de/datenschutz/">Datenschutz</a> - Powered by <a target="_blank" href="https://v2.vuepress.vuejs.org/">VuePress</a>',
    copyright:
      '&#169; 2025 Cedric Specht - <a target="_self" href="https://specht-labs.de/">Specht Labs</a>',
  },

  /**
   * @see https://theme-plume.vuejs.press/config/basic/#profile
   */
  profile: {
    avatar: "/images/specht-labs-rounded.png",
    name: "Specht Labs",
    description:
      "SpechtLabs is dedicated to building robust, scalable, and high-performance software.",
    // circle: true,
    location: "Hamburg, Germany",
    //    organization: 'foobar',
  },

  navbar,

  /**
   * 公告板
   * @see https://theme-plume.vuejs.press/guide/features/bulletin/
   */
  // bulletin: {
  //   layout: 'top-right',
  //   contentType: 'markdown',
  //   title: '公告板标题',
  //   content: '公告板内容',
  // },
});
