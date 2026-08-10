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
          if (id.includes('node_modules')) {
            if (id.includes('react') || id.includes('react-dom') || id.includes('react-router')) {
              return 'react-vendor';
            }
            if (id.includes('@xyflow/react') || id.includes('elkjs')) {
              return 'diagrams';
            }
            if (id.includes('lucide-react')) {
              return 'icons';
            }
            return 'vendor';
          }
        }
      }
    }
  },
  test: {
    include: ['test/**/*.test.ts']
  }
});
