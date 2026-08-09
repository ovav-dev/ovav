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
      
      social: [
        { icon: 'github', label: 'GitHub', href: 'https://github.com/ovav-dev/ovav-systems' },
      ],
      
      editLink: {
        baseUrl: 'https://github.com/ovav-dev/ovav-systems/edit/main/apps/docs/',
      },
      
      // ── Custom Theme Colors (synced with landing) ──
      customCss: [
        './src/assets/ovav.css',
      ],
      
      // ── Sidebar Configuration — Auto-generated from content ──
      sidebar: [
        {
          label: 'Getting Started',
          autogenerate: { directory: 'getting-started' },
        },
        {
          label: 'Core Concepts',
          autogenerate: { directory: 'core' },
        },
        {
          label: 'Reference',
          autogenerate: { directory: 'reference' },
        },
      ],
      
      // ── Head Customization ──
      head: [
        {
          tag: 'meta',
          attrs: {
            name: 'og:description',
            content: 'OVAV documentation — Professional Development Governance.',
          },
        },
        {
          tag: 'meta',
          attrs: {
            name: 'theme-color',
            content: '#10b981',
          },
        },
      ],
    }),
  ],
});
