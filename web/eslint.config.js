import js from '@eslint/js';
import tseslint from 'typescript-eslint';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';
import eslintReact from '@eslint-react/eslint-plugin';
import reactRefresh from 'eslint-plugin-react-refresh';

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
  // ESLint React recommended preset (replaces eslint-plugin-react)
  eslintReact.configs.recommended,
  {
    files: ['src/**/*.{ts,tsx}'],
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.es2022,
      },
      parserOptions: {
        projectService: true,
        tsconfigRootDir: import.meta.dirname,
      },
    },
    settings: {
      react: { version: '19.0' },
    },
    plugins: {
      'react-hooks': reactHooks,
      'react-refresh': reactRefresh,
      'local-rules': {
        rules: {
          'no-unstable-zustand-selectors': noUnstableZustandSelectors,
        },
      },
    },
    rules: {
      // ── Custom: Zustand ──
      'local-rules/no-unstable-zustand-selectors': 'error',

      // ── React hooks ──
      'react-hooks/rules-of-hooks': 'error',
      'react-hooks/exhaustive-deps': 'warn',

      // ── React best practices (migrated from eslint-plugin-react) ──
      // Most rules are already in the recommended preset; override severity here.
      '@eslint-react/no-array-index-key': 'warn',
      '@eslint-react/jsx-no-children-prop': 'error',
      '@eslint-react/no-direct-mutation-state': 'error',
      '@eslint-react/jsx-no-useless-fragment': 'warn',
      '@eslint-react/dom-no-missing-button-type': 'warn',
      '@eslint-react/dom-no-unknown-property': 'warn',

      // ── React Refresh (Vite HMR) ──
      'react-refresh/only-export-components': [
        'warn',
        { allowConstantExport: true },
      ],

      // ── TypeScript ──
      '@typescript-eslint/no-unused-vars': ['warn', {
        argsIgnorePattern: '^_',
        varsIgnorePattern: '^_',
        ignoreRestSiblings: true,
      }],
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/consistent-type-imports': ['warn', {
        prefer: 'type-imports',
        fixStyle: 'inline-type-imports',
      }],
      '@typescript-eslint/no-unused-expressions': 'warn',
      '@typescript-eslint/no-empty-function': 'warn',
      '@typescript-eslint/no-empty-object-type': 'warn',
      '@typescript-eslint/no-extra-non-null-assertion': 'error',
      '@typescript-eslint/no-non-null-asserted-optional-chain': 'error',
      '@typescript-eslint/no-unnecessary-type-assertion': 'warn',
      '@typescript-eslint/no-misused-promises': 'warn',
      '@typescript-eslint/no-floating-promises': 'warn',
      '@typescript-eslint/await-thenable': 'error',
      '@typescript-eslint/require-await': 'warn',

      // ── General JS best practices ──
      'no-console': ['warn', { allow: ['warn', 'error', 'info'] }],
      'no-debugger': 'error',
      'no-alert': 'warn',
      'no-eval': 'error',
      'no-implied-eval': 'error',
      'no-new-func': 'error',
      'no-new-wrappers': 'error',
      'no-self-compare': 'warn',
      'no-sequences': 'error',
      'no-throw-literal': 'error',
      'no-unmodified-loop-condition': 'warn',
      'no-unused-private-class-members': 'warn',
      'no-useless-call': 'warn',
      'no-useless-concat': 'warn',
      'no-useless-return': 'warn',
      'no-constant-binary-expression': 'warn',
      'no-duplicate-imports': 'warn',
      'no-useless-assignment': 'warn',
      'prefer-const': 'warn',
      'prefer-template': 'warn',
      eqeqeq: ['warn', 'smart'],
    },
  },
);
