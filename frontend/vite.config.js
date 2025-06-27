import { defineConfig } from 'vite'

export default defineConfig({
    root: 'src',

    build: {
        outDir: '../dist',
        emptyOutDir: true,
    },

    // AJOUTEZ CETTE SECTION
    server: {
        hmr: {
            protocol: 'ws',
            host: 'localhost'
        }
    }
})