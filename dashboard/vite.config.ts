import { defineConfig } from 'vite';
import react from '@vitejs/plugin-react';
import path from 'path';

export default defineConfig({
  plugins: [react()],
  resolve: {
    alias: {
      '@': path.resolve(__dirname, './src'),
    },
  },
  build: {
    rollupOptions: {
      output: {
        manualChunks: {
          // React core
          'vendor-react': ['react', 'react-dom', 'react-router-dom'],
          // Data fetching & state
          'vendor-query': ['@tanstack/react-query', 'axios'],
          // Charts
          'vendor-charts': ['chart.js', 'react-chartjs-2'],
        },
      },
    },
  },
  server: {
    port: 3000,
    proxy: {
      '/api': {
        target: 'http://localhost:8443',
        changeOrigin: true,
        secure: false,
      },
      '/ws': {
        target: 'ws://localhost:8443',
        ws: true,
        secure: false,
      },
    },
  },
});
