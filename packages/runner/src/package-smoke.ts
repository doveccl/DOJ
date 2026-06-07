import { DockerRunner } from './docker-runner'
import { judgePackage, type PackageJudgeInput } from './judge'

const runner = new DockerRunner()

const ccTesterDockerfile = [
  'FROM gcc:latest',
  'WORKDIR /workspace',
  'COPY main.cc /workspace/main.cc',
  'RUN g++ -std=c++20 -O2 -pipe -static -s main.cc -o main',
  'CMD ["/workspace/main"]'
].join('\n')

const limits = { timeMs: 2000, memoryBytes: 256 * 1024 * 1024, outputBytes: 1024 * 1024 }

async function judge(
  name: string,
  input: PackageJudgeInput,
  expect: (r: Awaited<ReturnType<typeof judgePackage>>) => boolean
) {
  const result = await judgePackage(runner, input)
  await runner.cleanup({ scopeId: input.scopeId })
  if (!expect(result)) {
    throw new Error(`case ${name}: unexpected result ${JSON.stringify(result)}`)
  }
  console.log({
    name,
    status: result.status,
    score: result.score,
    maxScore: result.maxScore,
    cases: result.cases.map((c) => `${c.caseIndex}:${c.status}:${c.score}`)
  })
}

// Default mode (no problem Dockerfile): engine pipes input -> B -> compares.
await judge(
  'default-ac',
  {
    scopeId: `pkg-default-ac-${crypto.randomUUID()}`,
    testerFiles: {
      Dockerfile: ccTesterDockerfile,
      'main.cc': '#include <cstdio>\nint main(){int a,b;scanf("%d %d",&a,&b);printf("%d\\n",a+b);}'
    },
    testCases: [
      { name: '1', input: '1 2\n', output: '3\n' },
      { name: '2', input: '4 5\n', output: '9\n' }
    ],
    limits
  },
  (r) => r.status === 'AC' && r.score === 100
)

// Default mode partial: B only succeeds when a+b is even (3 cases, middle WA).
await judge(
  'default-partial',
  {
    scopeId: `pkg-default-partial-${crypto.randomUUID()}`,
    testerFiles: {
      Dockerfile: ccTesterDockerfile,
      'main.cc':
        '#include <cstdio>\nint main(){int a,b;scanf("%d %d",&a,&b);int s=a+b;if(s%2)return 0;printf("%d\\n",s);}'
    },
    testCases: [
      { name: '1', input: '1 1\n', output: '2\n' },
      { name: '2', input: '1 2\n', output: '3\n' },
      { name: '3', input: '2 2\n', output: '4\n' }
    ],
    limits
  },
  (r) => r.status === 'WA' && r.score === 67 && r.cases.length === 3
)

// Default mode read-until-EOF: B reads an unknown number of ints until EOF.
// The engine closes B's stdin after feeding input, so this must NOT hang.
await judge(
  'default-eof',
  {
    scopeId: `pkg-default-eof-${crypto.randomUUID()}`,
    testerFiles: {
      Dockerfile: ccTesterDockerfile,
      'main.cc':
        '#include <cstdio>\nint main(){long long x,s=0;while(scanf("%lld",&x)==1)s+=x;printf("%lld\\n",s);}'
    },
    testCases: [{ name: '1', input: '1 2 3 4 5\n', output: '15\n' }],
    limits
  },
  (r) => r.status === 'AC' && r.score === 100
)

// Custom mode (interactor + checker): A talks to B and decides the verdict by
// exit code. Mirrors v4 testdata/a+b/spj: A generates a+b, reads B's answer.
await judge(
  'custom-interactor',
  {
    scopeId: `pkg-custom-${crypto.randomUUID()}`,
    problemFiles: {
      Dockerfile: [
        'FROM gcc:latest',
        'WORKDIR /src',
        'COPY judge.cc .',
        'RUN g++ -std=c++20 -O2 -static -s judge.cc -o judge',
        'CMD ["/src/judge"]'
      ].join('\n'),
      'judge.cc': [
        '#include <bits/stdc++.h>',
        'using namespace std;',
        'enum { AC, WA, PE, TLE, MLE, OLE, CE, RE, SE };',
        'int main(){',
        '  int cas = atoi(getenv("case")?getenv("case"):"0");',
        '  long long a = cas + 1, b = cas + 2;',
        '  cout << a << " " << b << endl;',
        '  long long got; cin >> got;',
        '  cerr << "want " << a+b << " got " << got << endl;',
        '  return a + b == got ? AC : WA;',
        '}'
      ].join('\n')
    },
    testerFiles: {
      Dockerfile: ccTesterDockerfile,
      'main.cc':
        '#include <cstdio>\nint main(){long long a,b;scanf("%lld %lld",&a,&b);printf("%lld\\n",a+b);}'
    },
    testCases: [
      { name: '1', input: '', output: '' },
      { name: '2', input: '', output: '' },
      { name: '3', input: '', output: '' }
    ],
    limits
  },
  (r) => r.status === 'AC' && r.score === 100
)

// Compile error in the submission (B) package -> CE.
await judge(
  'custom-ce',
  {
    scopeId: `pkg-ce-${crypto.randomUUID()}`,
    testerFiles: {
      Dockerfile: ccTesterDockerfile,
      'main.cc': 'int main(){ this is not valid c++ }'
    },
    testCases: [{ name: '1', input: '1 2\n', output: '3\n' }],
    limits
  },
  (r) => r.status === 'CE'
)

const cacheKey = `smoke-${crypto.randomUUID().replace(/-/g, '')}`
try {
  const files = {
    Dockerfile: 'FROM busybox:latest\nCMD ["true"]\n'
  }
  const firstBuild = await runner.buildPackage({
    scopeId: `pkg-cache-a-${crypto.randomUUID()}`,
    files,
    trusted: true,
    cacheKey
  })
  const secondBuild = await runner.buildPackage({
    scopeId: `pkg-cache-b-${crypto.randomUUID()}`,
    files,
    trusted: true,
    cacheKey
  })
  if (!firstBuild.ok || firstBuild.cached || !secondBuild.ok || !secondBuild.cached) {
    throw new Error(
      `expected package image cache miss then hit, got ${JSON.stringify({ firstBuild, secondBuild })}`
    )
  }
  console.log({
    name: 'package-cache',
    firstCached: firstBuild.cached,
    secondCached: secondBuild.cached
  })
} finally {
  await runner.cleanupPackageCache(cacheKey)
}

console.log('package engine smoke passed')
