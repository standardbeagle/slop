/**
 * Creating a sidebar enables you to:
 - create an ordered group of docs
 - render a sidebar for each doc of that group
 - provide next/previous navigation

 The sidebars can be generated from the filesystem, or explicitly defined here.

 Create as many sidebars as you want.
 */

// @ts-check

/** @type {import('@docusaurus/plugin-content-docs').SidebarsConfig} */
const sidebars = {
  tutorialSidebar: [
    'intro',
    {
      type: 'category',
      label: 'Getting Started',
      items: [
        'getting-started/installation',
        'getting-started/quick-start',
        'getting-started/first-script',
      ],
    },
    {
      type: 'category',
      label: 'Language Guide',
      items: [
        'language/spec',
        'language/syntax',
        'language/types',
        'language/control-flow',
        'language/functions',
        'language/modules',
      ],
    },
    {
      type: 'category',
      label: 'Built-in Functions',
      items: [
        'builtins/overview',
        'builtins/string',
        'builtins/math',
        'builtins/collections',
        'builtins/validation',
      ],
    },
    {
      type: 'category',
      label: 'Advanced Topics',
      items: [
        'advanced/agents',
        'advanced/mcp-integration',
        'advanced/safety',
        'advanced/performance',
      ],
    },
    {
      type: 'category',
      label: 'Examples',
      items: [
        'examples/chat-app',
        'examples/code-snippets',
        'examples/best-practices',
      ],
    },
    {
      type: 'category',
      label: 'API Reference',
      items: [
        'api/runtime',
        'api/evaluator',
        'api/parser',
      ],
    },
  ],
};

module.exports = sidebars;
