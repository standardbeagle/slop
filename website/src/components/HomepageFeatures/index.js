import React from 'react';
import clsx from 'clsx';
import styles from './styles.module.css';

const FeatureList = [
  {
    title: '🎯 Simple & Intuitive',
    description: (
      <>
        Python-like syntax that's easy to learn. Write AI workflows in minutes,
        not hours. No complex boilerplate or configuration required.
      </>
    ),
  },
  {
    title: '🔒 Safe by Design',
    description: (
      <>
        Built-in safety mechanisms prevent infinite loops, resource exhaustion,
        and unintended side effects. Perfect for production AI systems.
      </>
    ),
  },
  {
    title: '⚡ Fast & Efficient',
    description: (
      <>
        Lightweight Go runtime with streaming support. Execute complex agent
        workflows with minimal overhead and real-time responses.
      </>
    ),
  },
  {
    title: '🔌 MCP Integration',
    description: (
      <>
        Native Model Context Protocol support for tool calling and external
        service integration. Build powerful AI agents with ease.
      </>
    ),
  },
  {
    title: '📦 Modular Design',
    description: (
      <>
        Organize code into reusable modules. Share agents across projects
        with built-in dependency management and versioning.
      </>
    ),
  },
  {
    title: '🧪 Well-Tested',
    description: (
      <>
        200+ unit tests ensure reliability. Battle-tested patterns for
        common AI workflows and edge cases.
      </>
    ),
  },
];

function Feature({title, description}) {
  return (
    <div className={clsx('col col--4')}>
      <div className={clsx('feature', styles.feature)}>
        <h3>{title}</h3>
        <p>{description}</p>
      </div>
    </div>
  );
}

export default function HomepageFeatures() {
  return (
    <section className={styles.features}>
      <div className="container">
        <div className="row">
          {FeatureList.map((props, idx) => (
            <Feature key={idx} {...props} />
          ))}
        </div>
      </div>
    </section>
  );
}
