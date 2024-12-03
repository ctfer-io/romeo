import path from 'node:path'
import { fileURLToPath } from 'node:url'

import { fixupConfigRules, fixupPluginRules } from '@eslint/compat'
import { FlatCompat } from '@eslint/eslintrc'
import js from '@eslint/js'
import typescriptEslint from '@typescript-eslint/eslint-plugin'
import tsParser from '@typescript-eslint/parser'
import filenames from 'eslint-plugin-filenames'
import github from 'eslint-plugin-github'
import _import from 'eslint-plugin-import'
import noAsyncForeach from 'eslint-plugin-no-async-foreach'
import globals from 'globals'

import pluginHeader from 'eslint-plugin-header'
pluginHeader.rules.header.meta.schema = false

const __filename = fileURLToPath(import.meta.url)
const __dirname = path.dirname(__filename)
const compat = new FlatCompat({
    baseDirectory: __dirname,
    recommendedConfig: js.configs.recommended,
    allConfig: js.configs.all
})

export default [
    {
        ignores: ['eslint.config.mjs', '.github/**/*']
    },
    ...fixupConfigRules(
        compat.extends(
            'eslint:recommended',
            'plugin:@typescript-eslint/recommended',
            'plugin:@typescript-eslint/recommended-requiring-type-checking',
            'plugin:github/recommended',
            'plugin:github/typescript',
            'plugin:import/typescript'
        )
    ),
    {
        plugins: {
            '@typescript-eslint': fixupPluginRules(typescriptEslint),
            filenames: fixupPluginRules(filenames),
            github: fixupPluginRules(github),
            import: fixupPluginRules(_import),
            'no-async-foreach': noAsyncForeach
        },

        languageOptions: {
            parser: tsParser,
            ecmaVersion: 5,
            sourceType: 'module',

            globals: {
                ...globals.node
            },

            parserOptions: {
                project: './tsconfig.json'
            }
        },

        settings: {
            'import/resolver': {
                node: {
                    moduleDirectory: ['node_modules', 'src']
                },

                typescript: {}
            },
            'import/ignore': ['sinon', 'uuid', '@octokit/plugin-retry']
        },

        rules: {
            'filenames/match-regex': ['error', '^[a-z0-9-]+(\\.test)?$'],
            'i18n-text/no-en': 'off',

            'import/extensions': [
                'error',
                {
                    json: {}
                }
            ],

            'import/no-amd': 'error',
            'import/no-commonjs': 'error',
            'import/no-cycle': 'error',
            'import/no-dynamic-require': 'error',

            'import/no-extraneous-dependencies': [
                'error',
                {
                    devDependencies: true
                }
            ],

            'import/no-namespace': 'off',
            'import/no-unresolved': 'error',
            'import/no-webpack-loader-syntax': 'error',

            'import/order': [
                'error',
                {
                    alphabetize: {
                        order: 'asc'
                    },

                    'newlines-between': 'always'
                }
            ],

            'max-len': [
                'error',
                {
                    code: 120,
                    ignoreUrls: true,
                    ignoreStrings: true,
                    ignoreTemplateLiterals: true
                }
            ],

            'no-async-foreach/no-async-foreach': 'error',
            'no-sequences': 'error',
            'no-shadow': 'off',
            '@typescript-eslint/no-shadow': 'error',
            'one-var': ['error', 'never']
        }
    },
    {
        files: ['**/*.ts', '**/*.js'],

        rules: {
            '@typescript-eslint/no-explicit-any': 'off',
            '@typescript-eslint/no-unsafe-assignment': 'off',
            '@typescript-eslint/no-unsafe-member-access': 'off',
            '@typescript-eslint/no-var-requires': 'off',
            '@typescript-eslint/prefer-regexp-exec': 'off',
            '@typescript-eslint/require-await': 'off',
            '@typescript-eslint/restrict-template-expressions': 'off',
            'func-style': 'off'
        }
    }
]
