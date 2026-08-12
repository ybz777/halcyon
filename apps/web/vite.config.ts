import { defineConfig } from 'vite'
import react from '@vitejs/plugin-react'

export default defineConfig({
  plugins: [react()],
  server: {
    port: 5173,
    proxy: {
      // Gateway UpstreamService only matches /api/v1/organizations/ (with trailing segment).
      // POST /api/v1/organizations (create) is not routed by the gateway and returns 404.
      // Proxy organization + invitation paths directly to the organization service in local dev.
      '/api/v1/organizations': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
      '/api/v1/invitations': {
        target: 'http://localhost:8081',
        changeOrigin: true,
      },
      // Audit and notifications enforce GatewayTrust when X-Halcyon-* identity headers are
      // present. Proxy directly to those services in local dev (like organizations) so JWT
      // bearer auth works with X-Organization-ID tenant context.
      '/api/v1/audit-events': {
        target: 'http://localhost:8083',
        changeOrigin: true,
      },
      '/api/v1/notification-preferences': {
        target: 'http://localhost:8085',
        changeOrigin: true,
      },
      '/api': {
        target: 'http://localhost:8088',
        changeOrigin: true,
      },
    },
  },
})
