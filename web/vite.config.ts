import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';

export default defineConfig({
  base: '/static/',
  plugins: [react()],
  build: {
    outDir: '../internal/web/static',
    emptyOutDir: true,
    rollupOptions: {
      output: {
        entryFileNames: 'app.[hash].js',
        chunkFileNames: 'chunks/[name].[hash].js',
        assetFileNames: (assetInfo) => {
          if (assetInfo.name?.endsWith('.css')) return 'styles.[hash].css';
          return 'assets/[name].[hash][extname]';
        },
        manualChunks: (id: string) => {
          // elkjs is huge (~1.4MB) — split it out on its own
          if (id.includes('node_modules/elkjs')) {
            return 'elkjs';
          }
        }
      }
    }
  },
  test: {
    include: ['test/**/*.test.ts']
  }
});
