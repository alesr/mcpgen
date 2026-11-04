import { defineConfig } from 'astro/config';
import starlight from '@astrojs/starlight';
import starlightLinksValidator from 'starlight-links-validator';

// Project-page hosting on GitHub Pages: https://alesr.github.io/mcpgen/
export default defineConfig({
  site: 'https://alesr.github.io/mcpgen/',
  base: '/mcpgen',
  srcDir: './src',
  outDir: './dist',
  integrations: [
    starlight({
      title: 'mcpgen',
      description:
        'Deterministic Go codegen for spec-compliant MCP servers: generate, extend, and verify until green.',
      editLink: {
        baseUrl: 'https://github.com/alesr/mcpgen/edit/main/',
      },
      social: [
        {
          icon: 'github',
          label: 'GitHub',
          href: 'https://github.com/alesr/mcpgen',
        },
      ],
      sidebar: [
        { label: 'Home', link: '/' },
        { label: 'Quickstart', link: '/quickstart/' },
        { label: 'Tutorial: weather server', link: '/tutorial/' },
        {
          label: 'Concepts',
          items: [
            { label: 'Ownership model and regeneration', link: '/concepts/ownership-model/' },
            { label: 'Marker regions and `add`', link: '/concepts/marker-regions/' },
            { label: 'Project layout', link: '/concepts/project-layout/' },
            { label: 'check and conformance', link: '/concepts/check-and-conformance/' },
            { label: 'Security model', link: '/concepts/security-model/' },
          ],
        },
        {
          label: 'How-to guides',
          items: [
            { label: 'Add tools, prompts, resources', link: '/guides/add-capabilities/' },
            { label: 'Transports, protocol, features', link: '/guides/transports-and-features/' },
            { label: 'Regenerate safely', link: '/guides/regenerate-safely/' },
            { label: 'Run check in CI', link: '/guides/check-in-ci/' },
            { label: 'Drive mcpgen from agents', link: '/guides/drive-from-agents/' },
            { label: 'Deploy with Docker', link: '/guides/deploy-docker/' },
          ],
        },
        {
          label: 'Reference',
          items: [
            { label: 'CLI commands and flags', link: '/reference/generated/cli/' },
            { label: 'mcpgen.json', link: '/reference/generated/config/' },
            { label: 'Machine interface (JSON, exit codes)', link: '/reference/generated/machine-interface/' },
            { label: 'Generated files and check stages', link: '/reference/generated/project-files/' },
          ],
        },
        {
          label: 'Project',
          items: [
            { label: 'Contributing and architecture', link: '/project/contributing/' },
            { label: 'FAQ and troubleshooting', link: '/project/faq/' },
          ],
        },
      ],
      plugins: [
        starlightLinksValidator({
          errorOnFallbackPages: false,
        }),
      ],
    }),
  ],
});
