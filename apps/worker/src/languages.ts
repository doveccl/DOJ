import { languageDescriptors } from '@doj/shared/languages'

export interface LanguageRuntime {
  id: string
  name: string
  sourceFile: string
  dockerfile(sourceFile: string): string
  command: string[]
}

export const languageList: LanguageRuntime[] = [
  {
    ...languageDescriptors.find((language) => language.id === 'sh')!,
    dockerfile: (sourceFile) =>
      [
        'FROM alpine:latest',
        'WORKDIR /work',
        `COPY ${sourceFile} /work/${sourceFile}`,
        `RUN chmod +x /work/${sourceFile}`,
        `CMD ["/work/${sourceFile}"]`
      ].join('\n'),
    command: []
  },
  {
    ...languageDescriptors.find((language) => language.id === 'py')!,
    dockerfile: (sourceFile) =>
      [
        'FROM python:alpine',
        'WORKDIR /work',
        `COPY ${sourceFile} /work/${sourceFile}`,
        `CMD ["python", "/work/${sourceFile}"]`
      ].join('\n'),
    command: []
  }
]

export const languages = new Map(languageList.map((language) => [language.id, language]))

export function getLanguage(id: string) {
  const language = languages.get(id)
  if (!language) throw new Error(`unsupported language: ${id}`)
  return language
}
