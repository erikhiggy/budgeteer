import react from '@vitejs/plugin-react'
import { defineConfig } from 'vitest/config'

// Must match the Go server's default --addr.
const apiTarget = 'http://127.0.0.1:7420'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    strictPort: true,
    proxy: { '/api': apiTarget },
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
  },
  test: {
    include: ['src/**/*.test.{ts,tsx}'],
  },
})
