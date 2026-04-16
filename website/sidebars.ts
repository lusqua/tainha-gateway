import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docsSidebar: [
    'getting-started',
    'configuration',
    {
      type: 'category',
      label: 'Authentication',
      link: {type: 'doc', id: 'authentication/index'},
      items: [
        'authentication/local-jwt',
        'authentication/delegation',
      ],
    },
    'response-mapping',
    'sse',
    'observability',
  ],
};

export default sidebars;
