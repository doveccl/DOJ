export interface JudgeLanguageConfig {
  id: string
  name: string
  source: string
  dockerfile: string
  sort: number
}

export const cppDockerfile = [
  'FROM gcc:14',
  'WORKDIR /app',
  'COPY main.cc /app/main.cc',
  'RUN g++ -std=c++20 -O2 -pipe -static -s -o /app/main /app/main.cc',
  'CMD ["/app/main"]'
].join('\n')

export const defaultLanguageConfigs = [
  {
    id: 'cpp',
    name: 'C/C++',
    source: 'main.cc',
    dockerfile: cppDockerfile,
    sort: 10
  }
] as const satisfies readonly JudgeLanguageConfig[]

export const languageDescriptors = defaultLanguageConfigs.map(({ id, name, source }) => ({
  id,
  name,
  source
}))
export type LanguageId = (typeof defaultLanguageConfigs)[number]['id']
