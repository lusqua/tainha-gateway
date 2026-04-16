import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import Link from '@docusaurus/Link';

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header style={{
      padding: '4rem 0',
      textAlign: 'center',
      position: 'relative',
      overflow: 'hidden',
    }}>
      <div className="container">
        <h1 style={{fontSize: '3rem'}}>{siteConfig.title}</h1>
        <p style={{fontSize: '1.4rem', opacity: 0.8}}>{siteConfig.tagline}</p>
        <div style={{display: 'flex', gap: '1rem', justifyContent: 'center', marginTop: '2rem'}}>
          <Link
            className="button button--primary button--lg"
            to="/docs/getting-started">
            Get Started
          </Link>
          <Link
            className="button button--secondary button--lg"
            href="https://github.com/lusqua/tainha-gateway">
            GitHub
          </Link>
        </div>
      </div>
    </header>
  );
}

function Features() {
  const features = [
    {
      title: 'Simple Configuration',
      description: 'Define routes, auth, and mappings in a single YAML file. No code required to set up your gateway.',
    },
    {
      title: 'JWT Authentication',
      description: 'Built-in JWT validation with HS256, or delegate to your own auth service. Protect routes with a single flag.',
    },
    {
      title: 'Response Mapping',
      description: 'Enrich API responses by aggregating data from multiple services automatically with parallel requests.',
    },
    {
      title: 'SSE Support',
      description: 'Native Server-Sent Events passthrough for real-time streaming between clients and backends.',
    },
    {
      title: 'Auth Delegation',
      description: 'Bring your own auth service. Tainha validates tokens by calling your service — use any auth strategy you want.',
    },
    {
      title: 'Lightweight',
      description: 'Single binary, minimal dependencies. Built with Go for performance and simplicity.',
    },
  ];

  return (
    <section style={{padding: '4rem 0'}}>
      <div className="container">
        <div className="row">
          {features.map((feature, idx) => (
            <div key={idx} className="col col--4" style={{marginBottom: '2rem'}}>
              <h3>{feature.title}</h3>
              <p>{feature.description}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
}

export default function Home(): React.JSX.Element {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout title={siteConfig.title} description={siteConfig.tagline}>
      <HomepageHeader />
      <main>
        <Features />
      </main>
    </Layout>
  );
}
