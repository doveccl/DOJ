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
    id: 'c',
    name: 'C',
    enabled: true,
    sourceFile: 'main.c',
    dockerfile: (sourceFile: string) =>
      [
        'FROM gcc:latest',
        'WORKDIR /workspace',
        `COPY ${sourceFile} /workspace/${sourceFile}`,
        `RUN gcc -std=c17 -O2 -pipe -static -s ${sourceFile} -o main`,
        'CMD ["/workspace/main"]'
      ].join('\n'),
    command: ['/workspace/main'] as string[],
    sortOrder: 10
  },
  {
    id: 'cpp',
    name: 'C++',
    enabled: true,
    sourceFile: 'main.cpp',
    dockerfile: (sourceFile: string) =>
      [
        'FROM gcc:latest',
        'WORKDIR /workspace',
        `COPY ${sourceFile} /workspace/${sourceFile}`,
        `RUN g++ -std=c++20 -O2 -pipe -static -s ${sourceFile} -o main`,
        'CMD ["/workspace/main"]'
      ].join('\n'),
    command: ['/workspace/main'] as string[],
    sortOrder: 20
  }
] as const

export const languageDescriptors = defaultLanguageConfigs.map(({ id, name, sourceFile }) => ({
  id,
  name,
  sourceFile
}))
export type LanguageId = (typeof defaultLanguageConfigs)[number]['id']
