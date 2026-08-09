import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';

// https://starlight.astro.build/
export default defineConfig({
  output: 'static',
  site: 'https://docs.ovav.dev',
  integrations: [
    starlight({
      title: 'OVAV Docs',
      description: 'OVAV — Professional Development Governance. Complete documentation.',
      logo: {
        src: './src/assets/ovav-logo.svg',
      },
      social: {
        github: 'https://github.com/ovav-ai/ovav-docs',
      },
      editLink: {
        baseUrl: 'https://github.com/ovav-ai/ovav-docs/edit/main/',
      },
      sidebar: [
        {
          label: 'Getting Started',
          items: [
            { label: 'Introduction', slug: 'getting-started/intro' },
            { label: 'Quickstart', slug: 'getting-started/quickstart' },
            { label: 'Installation', slug: 'getting-started/installation' },
          ],
        },
        {
          label: 'Core Concepts',
          items: [
            { label: 'Architecture', slug: 'core/architecture' },
            { label: 'Professional Profiles', slug: 'core/profiles' },
            { label: 'Governance Model', slug: 'core/governance' },
            { label: 'Multi-Model Runtime', slug: 'core/multi-model' },
          ],
        },
        {
          label: 'Guides',
          items: [
            { label: 'First Profile Setup', slug: 'guides/first-profile' },
            { label: 'Tailor Composer', slug: 'guides/tailor' },
            { label: 'Vault Encryption', slug: 'guides/vault' },
            { label: 'cPanel Administration', slug: 'guides/cpanel' },
          ],
        },
        {
          label: 'Reference',
          items: [
            { label: 'CLI Commands', slug: 'reference/cli' },
            { label: 'API Reference', slug: 'reference/api' },
            { label: 'Security', slug: 'reference/security' },
            { label: 'Configuration', slug: 'reference/configuration' },
          ],
        },
      ],
      customCss: [
        './src/assets/ovav.css',
      ],
      head: [
        {
          tag: 'meta',
          attrs: {
            name: 'og:description',
            content: 'OVAV documentation — Professional Development Governance.',
          },
        },
      ],
    }),
  ],
  vite: {
    ssr: {
      noExternal: ['@astrojs/starlight'],
    },
  },
});
