export interface JudgeLanguageConfig {
  id: string
  name: string
  enabled: boolean
  sourceFile: string
  dockerfile: string
  command: string[]
  sortOrder: number
}

export const defaultLanguageConfigs = [
  {
    id: 'sh',
    name: 'Shell',
    enabled: true,
    sourceFile: 'main.sh',
    dockerfile: (sourceFile: string) =>
      [
        'FROM alpine:latest',
        'WORKDIR /workspace',
        `COPY ${sourceFile} /workspace/${sourceFile}`,
        `RUN chmod +x /workspace/${sourceFile}`,
        `CMD ["/workspace/${sourceFile}"]`
      ].join('\n'),
    command: ['/workspace/main.sh'] as string[],
    sortOrder: 10
  },
  {
    id: 'py',
    name: 'Python',
    enabled: true,
    sourceFile: 'main.py',
    dockerfile: (sourceFile: string) =>
      [
        'FROM python:latest',
        'WORKDIR /workspace',
        `COPY ${sourceFile} /workspace/${sourceFile}`,
        `CMD ["python3", "/workspace/${sourceFile}"]`
      ].join('\n'),
    command: ['python3', '/workspace/main.py'] as string[],
    sortOrder: 20
  }
] as const

export const languageDescriptors = defaultLanguageConfigs.map(({ id, name, sourceFile }) => ({
  id,
  name,
  sourceFile
}))
export type LanguageId = (typeof defaultLanguageConfigs)[number]['id']
