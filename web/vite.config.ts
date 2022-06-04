import { defineConfig } from 'vite';
import { nodeResolve } from '@rollup/plugin-node-resolve';
import commonjs from '@rollup/plugin-commonjs';
import solidPlugin from 'vite-plugin-solid';

export default defineConfig({
  plugins: [solidPlugin(), nodeResolve(), commonjs()],
  build: {
    target: 'esnext',
    polyfillDynamicImport: false,
  },
});
