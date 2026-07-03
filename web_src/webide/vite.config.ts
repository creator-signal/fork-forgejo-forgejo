import { defineConfig } from 'vite'

// Config adapted from the monaco-vscode-api demo. Output is vendored into
// Forgejo's public/webide and served at /webide/ by the full-screen IDE shell.
export default defineConfig({
  base: '/assets/webide/',
  build: {
    target: 'esnext',
    outDir: '../../public/assets/webide',
    emptyOutDir: true
  },
  worker: {
    format: 'es'
  },
  plugins: [
    {
      // monaco-vscode-api ships CSS meant to be imported as a string
      name: 'load-vscode-css-as-string',
      enforce: 'pre',
      async resolveId(source, importer, options) {
        const resolved = await this.resolve(source, importer, options)
        if (resolved == null) return undefined
        if (resolved.id.match(/node_modules\/(@codingame\/monaco-vscode|vscode|monaco-editor).*\.css$/)) {
          return { ...resolved, id: resolved.id + '?inline' }
        }
        return undefined
      }
    }
  ],
  optimizeDeps: {
    include: [
      '@codingame/monaco-vscode-api',
      '@codingame/monaco-vscode-api/extensions',
      '@codingame/monaco-vscode-api/monaco',
      'vscode/localExtensionHost'
    ]
  }
})
