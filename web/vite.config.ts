import { defineConfig } from 'vite'
import { svelte } from '@sveltejs/vite-plugin-svelte'

// The SPA is served by the Threadfin binary at /ui/ (see src/webserver.go).
// `npm run dev` proxies API traffic to a locally running Threadfin instance.
export default defineConfig({
  base: '/ui/',
  plugins: [svelte()],
  build: {
    outDir: '../webui/dist',
    emptyOutDir: true,
  },
  server: {
    proxy: {
      '/data/': {
        target: 'ws://localhost:34400',
        ws: true,
      },
      '/web/': 'http://localhost:34400',
      '/images/': 'http://localhost:34400',
      '/data_images/': 'http://localhost:34400',
      '/download/': 'http://localhost:34400',
      '/api/': 'http://localhost:34400',
    },
  },
})
