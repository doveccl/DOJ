export interface LanguageRuntime {
  id: string
  sourceFile: string
  dockerfile(sourceFile: string): string
  command: string[]
}

export const languages = new Map<string, LanguageRuntime>([
  [
    'sh',
    {
      id: 'sh',
      sourceFile: 'main.sh',
      dockerfile: (sourceFile) =>
        [
          'FROM alpine:latest',
          'WORKDIR /work',
          `COPY ${sourceFile} /work/${sourceFile}`,
          `RUN chmod +x /work/${sourceFile}`,
          `CMD ["/work/${sourceFile}"]`
        ].join('\n'),
      command: []
    }
  ]
])

export function getLanguage(id: string) {
  const language = languages.get(id)
  if (!language) throw new Error(`unsupported language: ${id}`)
  return language
}
