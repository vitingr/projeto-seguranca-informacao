import nextVitals from 'eslint-config-next/core-web-vitals'
import js from '@eslint/js'
import typescript from '@typescript-eslint/eslint-plugin'
import parser from '@typescript-eslint/parser'
import prettier from 'eslint-plugin-prettier'
import importHelpers from 'eslint-plugin-import-helpers'
import unusedImports from 'eslint-plugin-unused-imports'
import perfectionist from 'eslint-plugin-perfectionist'
import svgJsx from 'eslint-plugin-svg-jsx'

// eslint-disable-next-line import/no-anonymous-default-export
export default [
  ...nextVitals,
  {
    ...js.configs.recommended,
    ignores: ['node_modules/**', '.next/*'],
    files: ['**/*.{js,ts,jsx,tsx}'],
    languageOptions: {
      parser,
      parserOptions: {
        ecmaVersion: 'latest',
        sourceType: 'module'
      }
    },
    plugins: {
      '@typescript-eslint': typescript,
      prettier,
      'import-helpers': importHelpers,
      'unused-imports': unusedImports,
      perfectionist,
      'svg-jsx': svgJsx
    },
    rules: {
      'no-useless-constructor': 'off',
      'no-use-before-define': 'off',
      'space-before-function-paren': 'off',
      camelcase: 'off',
      'prettier/prettier': 'error',
      // User content is rendered as text only (005 D-6, S-3).
      'react/no-danger': 'error',
      'unused-imports/no-unused-imports': 'error',
      'svg-jsx/camel-case-dash': 'error',
      'svg-jsx/camel-case-colon': 'error',
      'svg-jsx/no-style-string': 'error',
      'perfectionist/sort-interfaces': 'error',
      'perfectionist/sort-jsx-props': [
        'error',
        {
          type: 'natural',
          order: 'asc',
          groups: ['multiline', 'unknown', 'shorthand']
        }
      ],
      '@typescript-eslint/consistent-type-imports': [
        'error',
        {
          prefer: 'type-imports',
          disallowTypeAnnotations: false
        }
      ],
      '@typescript-eslint/no-use-before-define': 'off',
      '@typescript-eslint/no-empty-interface': 'off',
      '@typescript-eslint/no-unused-vars': [
        'error',
        {
          vars: 'all',
          varsIgnorePattern: '^_',
          args: 'none',
          argsIgnorePattern: '^_'
        }
      ],
      'import-helpers/order-imports': [
        'warn',
        {
          newlinesBetween: 'always',
          groups: [
            ['module', '/^@ant/', '/^@fullstory/'],
            '/^@/',
            ['parent', 'sibling', 'index']
          ],
          alphabetize: { order: 'asc', ignoreCase: true }
        }
      ],
      'sort-imports': [
        'error',
        {
          ignoreCase: false,
          ignoreDeclarationSort: true,
          ignoreMemberSort: false,
          memberSyntaxSortOrder: ['none', 'all', 'single', 'multiple'],
          allowSeparatedGroups: false
        }
      ]
    }
  }
]
