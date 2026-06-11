import js from '@eslint/js'
import eslintConfigPrettier from 'eslint-config-prettier/flat'
import globals from 'globals'
import tseslint from 'typescript-eslint'
import vue from 'eslint-plugin-vue'

export default tseslint.config(
  {
    ignores: ['node_modules/**', 'dist/**', 'apps/web/dist/**', 'drizzle/**', 'bun.lock']
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...vue.configs['flat/recommended'],
  {
    files: ['**/*.{ts,vue}'],
    languageOptions: {
      globals: {
        ...globals.browser,
        ...globals.node,
        computed: 'readonly',
        defineStore: 'readonly',
        h: 'readonly',
        nextTick: 'readonly',
        onMounted: 'readonly',
        onUnmounted: 'readonly',
        reactive: 'readonly',
        ref: 'readonly',
        RouterLink: 'readonly',
        useRoute: 'readonly',
        useRouter: 'readonly',
        watch: 'readonly'
      },
      parserOptions: {
        parser: tseslint.parser
      }
    },
    rules: {
      '@typescript-eslint/no-explicit-any': 'off',
      '@typescript-eslint/no-unused-vars': [
        'warn',
        {
          argsIgnorePattern: '^_',
          varsIgnorePattern: '^_'
        }
      ],
      'vue/multi-word-component-names': 'off',
      'vue/no-v-html': 'off'
    }
  },
  eslintConfigPrettier
)
