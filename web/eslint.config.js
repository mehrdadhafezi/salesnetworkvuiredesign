import js from '@eslint/js'
import reactHooks from 'eslint-plugin-react-hooks'
import globals from 'globals'
import tseslint from 'typescript-eslint'

export default tseslint.config(
  // Generated code is never linted or edited; CI regenerates it.
  { ignores: ['dist', 'src/generated/**'] },
  {
    files: ['**/*.{ts,tsx}'],
    extends: [js.configs.recommended, ...tseslint.configs.strict],
    languageOptions: { globals: globals.browser },
    plugins: { 'react-hooks': reactHooks },
    rules: {
      ...reactHooks.configs.recommended.rules,
      // All HTTP goes through the generated client in src/shared/api.
      'no-restricted-globals': ['error', { name: 'fetch', message: 'Use the generated API client (src/shared/api).' }],
      'no-restricted-syntax': [
        'error',
        { selector: "MemberExpression[object.name='window'][property.name='fetch']", message: 'Use the generated API client (src/shared/api).' },
        { selector: "MemberExpression[object.name='globalThis'][property.name='fetch']", message: 'Use the generated API client (src/shared/api).' },
      ],
      'no-restricted-imports': [
        'error',
        { patterns: [{ group: ['**/generated/api/*'], message: 'Import contract types through src/shared/api.' }] },
      ],
    },
  },
  {
    // The API adapter is the only place that wires the network and the generated types.
    files: ['src/shared/api/**'],
    rules: { 'no-restricted-globals': 'off', 'no-restricted-syntax': 'off', 'no-restricted-imports': 'off' },
  },
)
