// @ts-check
// Note: type annotations allow type checking and IDEs autocompletion

const lightCodeTheme = require('prism-react-renderer').themes.github;
const darkCodeTheme = require('prism-react-renderer').themes.dracula;

/** @type {import('@docusaurus/types').Config} */
const config = {
  title: 'SLOP',
  tagline: 'Structured Language for Orchestrating Prompts',
  favicon: 'img/favicon.ico',

  // Set the production url of your site here
  url: 'https://standardbeagle.github.io',
  // Set the /<baseUrl>/ pathname under which your site is served
  baseUrl: '/slop/',

  // GitHub pages deployment config
  organizationName: 'standardbeagle',
  projectName: 'slop',
  deploymentBranch: 'gh-pages',
  trailingSlash: false,

  onBrokenLinks: 'warn',
  onBrokenMarkdownLinks: 'warn',

  i18n: {
    defaultLocale: 'en',
    locales: ['en'],
  },

  presets: [
    [
      'classic',
      /** @type {import('@docusaurus/preset-classic').Options} */
      ({
        docs: {
          sidebarPath: require.resolve('./sidebars.js'),
          editUrl: 'https://github.com/standardbeagle/slop/tree/main/website/',
        },
        blog: {
          showReadingTime: true,
          editUrl: 'https://github.com/standardbeagle/slop/tree/main/website/',
        },
        theme: {
          customCss: require.resolve('./src/css/custom.css'),
        },
      }),
    ],
  ],

  themeConfig:
    /** @type {import('@docusaurus/preset-classic').ThemeConfig} */
    ({
      image: 'img/slop-social-card.jpg',
      navbar: {
        title: 'SLOP',
        logo: {
          alt: 'SLOP Logo',
          src: 'img/logo.svg',
        },
        items: [
          {
            type: 'docSidebar',
            sidebarId: 'tutorialSidebar',
            position: 'left',
            label: 'Docs',
          },
          {to: '/blog', label: 'Blog', position: 'left'},
          {
            href: 'https://github.com/standardbeagle/slop',
            label: 'GitHub',
            position: 'right',
          },
        ],
      },
      footer: {
        style: 'dark',
        links: [
          {
            title: 'Docs',
            items: [
              {
                label: 'Getting Started',
                to: '/docs/intro',
              },
              {
                label: 'Language Specification',
                to: '/docs/spec',
              },
              {
                label: 'Examples',
                to: '/docs/examples',
              },
            ],
          },
          {
            title: 'Community',
            items: [
              {
                label: 'GitHub Issues',
                href: 'https://github.com/standardbeagle/slop/issues',
              },
              {
                label: 'Discussions',
                href: 'https://github.com/standardbeagle/slop/discussions',
              },
            ],
          },
          {
            title: 'More',
            items: [
              {
                label: 'Blog',
                to: '/blog',
              },
              {
                label: 'GitHub',
                href: 'https://github.com/standardbeagle/slop',
              },
            ],
          },
        ],
        copyright: `Copyright © ${new Date().getFullYear()} SLOP Project. Built with Docusaurus.`,
      },
      prism: {
        theme: lightCodeTheme,
        darkTheme: darkCodeTheme,
        additionalLanguages: ['go', 'python', 'bash'],
      },
      algolia: {
        // The application ID provided by Algolia
        appId: 'YOUR_APP_ID',
        // Public API key: it is safe to commit it
        apiKey: 'YOUR_SEARCH_API_KEY',
        indexName: 'slop',
        // Optional
        contextualSearch: true,
      },
    }),
};

module.exports = config;
