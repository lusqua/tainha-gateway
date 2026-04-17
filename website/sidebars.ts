import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docsSidebar: [
    'getting-started',
    {
      type: 'category',
      label: 'Configuration',
      link: {type: 'doc', id: 'configuration/index'},
      items: [
        'configuration/server',
        'configuration/auth',
        'configuration/rate-limiting',
        'configuration/resilience',
        'configuration/telemetry',
        'configuration/routes',
      ],
    },
    {
      type: 'category',
      label: 'Authentication',
      link: {type: 'doc', id: 'authentication/index'},
      items: [
        'authentication/local-jwt',
        'authentication/jwks',
        'authentication/delegation',
      ],
    },
    {
      type: 'category',
      label: 'Response Mapping',
      link: {type: 'doc', id: 'response-mapping/index'},
      items: [
        'response-mapping/how-it-works',
        'response-mapping/configuration',
        'response-mapping/examples',
      ],
    },
    'sse',
    'websocket',
    'resilience',
    'observability',
  ],
};

export default sidebars;
