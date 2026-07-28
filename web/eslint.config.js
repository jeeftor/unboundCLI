import js from '@eslint/js';
import tseslint from 'typescript-eslint';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';

// Custom rule: flag Zustand selectors that return new references
import noUnstableZustandSelectors from './eslint-rules/no-unstable-zustand-selectors.js';

export default tseslint.config(
  {
    ignores: [
      'dist/**',
      'node_modules/**',
      'internal/web/static/**',
      'eslint-rules/**',
    ],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  {
    files: ['src/**/*.{ts,tsx}'],
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.es2022,
      },
    },
    plugins: {
      'react-hooks': reactHooks,
      'local-rules': {
        rules: {
          'no-unstable-zustand-selectors': noUnstableZustandSelectors,
        },
      },
    },
    rules: {
      'local-rules/no-unstable-zustand-selectors': 'error',
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'warn',
      '@typescript-eslint/no-unused-vars': ['warn', { argsIgnorePattern: '^_' }],
      '@typescript-eslint/no-explicit-any': 'off',
    },
  },
);
