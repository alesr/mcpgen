import { defineCollection } from 'astro:content';
import { glob } from 'astro/loaders';
import { docsSchema, i18nSchema } from '@astrojs/starlight/schema';

export const collections = {
  docs: defineCollection({
    loader: glob({ pattern: '**/*.{md,mdx}', base: './src/content/docs' }),
    schema: docsSchema(),
  }),
  // Defined (empty) so Starlight's i18n lookup does not warn about a missing
  // collection; the site is English-only.
  i18n: defineCollection({
    loader: glob({ pattern: '**/*.{json,yaml}', base: './src/content/i18n' }),
    schema: i18nSchema(),
  }),
};
