import React from 'react';
import clsx from 'clsx';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import Layout from '@theme/Layout';
import HomepageFeatures from '@site/src/components/HomepageFeatures';

import styles from './index.module.css';

function HomepageHeader() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <header className={clsx('hero hero--primary', styles.heroBanner)}>
      <div className="container">
        <h1 className="hero__title">{siteConfig.title}</h1>
        <p className="hero__subtitle">{siteConfig.tagline}</p>
        <p className={styles.heroDescription}>
          A sandboxed execution environment for LLM-generated code.
          Scripts pause mid-run; the code, call stack, and memory are saved
          as editable JSON — fix anything, then resume with no work lost.
        </p>
        <div className={styles.buttons}>
          <Link
            className="button button--secondary button--lg"
            to="/docs/intro">
            Get Started - 5min ⏱️
          </Link>
          <Link
            className="button button--outline button--lg margin-left--md"
            to="/docs/examples/chat-app">
            View Examples 📚
          </Link>
        </div>
        <div className={styles.codeExample}>
          <pre>
            <code>{`# Chain MCP calls, pause, edit, resume
repos = github.search(query: "mcp servers")
pause("after_fetch")   # state saved as editable JSON
result = llm.call(prompt: "Summarize: " + json_stringify(repos), schema: {summary: string})
emit(result.summary)`}</code>
          </pre>
        </div>
      </div>
    </header>
  );
}

export default function Home() {
  const {siteConfig} = useDocusaurusContext();
  return (
    <Layout
      title={`Hello from ${siteConfig.title}`}
      description="SLOP - Structured Language for Orchestrating Prompts">
      <HomepageHeader />
      <main>
        <HomepageFeatures />
      </main>
    </Layout>
  );
}
