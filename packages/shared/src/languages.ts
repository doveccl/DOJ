export const languageDescriptors = [
  { id: 'sh', name: 'Shell', sourceFile: 'main.sh' },
  { id: 'py', name: 'Python', sourceFile: 'main.py' }
] as const

export type LanguageId = (typeof languageDescriptors)[number]['id']
