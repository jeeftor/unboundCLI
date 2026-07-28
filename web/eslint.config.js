import js from '@eslint/js';
import tseslint from 'typescript-eslint';
import globals from 'globals';
import reactHooks from 'eslint-plugin-react-hooks';
import react from 'eslint-plugin-react';
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
      react,
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

      // ── React best practices ──
      'react/jsx-key': 'error',                    // missing key in .map()
      'react/jsx-no-script-url': 'error',          // href="javascript:..."
      'react/jsx-no-duplicate-props': 'error',     // same prop twice
      'react/jsx-no-useless-fragment': 'warn',     // <></> with single child
      'react/no-array-index-key': 'warn',          // key={index} (can cause bugs)
      'react/no-children-prop': 'error',           // <Component children={...} />
      'react/no-danger': 'warn',                   // dangerouslySetInnerHTML
      'react/no-deprecated': 'error',              // deprecated React APIs
      'react/no-direct-mutation-state': 'error',   // this.state = ...
      'react/no-find-dom-node': 'error',           // findDOMNode (deprecated)
      'react/no-is-mounted': 'error',              // isMounted (deprecated)
      'react/no-render-return-value': 'error',     // using ReactDOM.render return
      'react/no-string-refs': 'error',             // ref="foo" (legacy)
      'react/no-unknown-property': 'warn',         // typos in DOM props
      'react/no-unescaped-entities': 'warn',       // unescaped quotes in JSX text
      'react/react-in-jsx-scope': 'off',           // not needed with new JSX transform
      'react/prop-types': 'off',                   // using TS, not propTypes
      'react/self-closing-comp': 'warn',           // <Foo></Foo> → <Foo />
      'react/wrap-multilines': 'off',              // stylistic, skip

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
