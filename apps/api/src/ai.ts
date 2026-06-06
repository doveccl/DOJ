import { getRuntimeSettings } from './settings'

export interface CoachingInput {
  status: string
  message: string
  languageId: string
  sourceCode: string
}

export interface CoachingOutput {
  model: string
  responseMarkdown: string
}

export async function createCoachingResponse(input: CoachingInput): Promise<CoachingOutput> {
  const settings = await getRuntimeSettings()
  if (settings.aiProvider === 'openai') return createOpenAiCoachingResponse(input, settings)

  return {
    model: 'local-rules',
    responseMarkdown: createLocalCoachingResponse(input.status, input.message)
  }
}

async function createOpenAiCoachingResponse(
  input: CoachingInput,
  settings: { aiApiKey: string; aiBaseUrl: string; aiModel: string }
): Promise<CoachingOutput> {
  if (!settings.aiApiKey) throw new Error('AI API key is required when AI provider is openai')

  const baseUrl = settings.aiBaseUrl.replace(/\/+$/, '')
  const response = await fetch(`${baseUrl}/responses`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      authorization: `Bearer ${settings.aiApiKey}`
    },
    body: JSON.stringify({
      model: settings.aiModel,
      reasoning: { effort: 'low' },
      input: [
        {
          role: 'developer',
          content: [
            'You are an online judge coach.',
            'Help the student debug a non-AC submission without revealing hidden tests, official solutions, or full corrected code.',
            'Use concise Markdown with likely causes, inspection steps, and small hints.'
          ].join('\n')
        },
        {
          role: 'user',
          content: [
            `Status: ${input.status}`,
            `Language: ${input.languageId}`,
            input.message ? `Judge message:\n${input.message}` : 'Judge message: empty',
            'Source code:',
            '```',
            input.sourceCode.slice(0, 20_000),
            '```'
          ].join('\n\n')
        }
      ]
    })
  })

  const body = (await response.json()) as {
    output_text?: string
    error?: { message?: string }
    output?: Array<{ content?: Array<{ text?: string }> }>
  }

  if (!response.ok) {
    throw new Error(body.error?.message ?? `OpenAI Responses API failed: ${response.status}`)
  }

  return {
    model: settings.aiModel,
    responseMarkdown: body.output_text ?? readOutputText(body) ?? ''
  }
}

function readOutputText(body: { output?: Array<{ content?: Array<{ text?: string }> }> }) {
  return body.output
    ?.flatMap((item) => item.content ?? [])
    .map((item) => item.text)
    .filter((text): text is string => !!text)
    .join('\n')
}

function createLocalCoachingResponse(status: string, message: string) {
  switch (status) {
    case 'CE':
      return [
        '### Compile Error',
        '',
        'Your code did not compile. Start by reading the first compiler error, then check syntax, missing imports, and language version assumptions.',
        '',
        message ? `Compiler output:\n\n\`\`\`text\n${message.trim()}\n\`\`\`` : ''
      ].join('\n')
    case 'RE':
      return [
        '### Runtime Error',
        '',
        'Your program crashed or exited with a non-zero status. Check array bounds, division by zero, failed parsing, and assumptions about empty input.',
        '',
        message ? `Runtime output:\n\n\`\`\`text\n${message.trim()}\n\`\`\`` : ''
      ].join('\n')
    case 'TLE':
      return '### Time Limit Exceeded\n\nYour solution ran too long. Revisit the algorithmic complexity and look for loops that may not terminate.'
    case 'MLE':
      return '### Memory Limit Exceeded\n\nYour solution used too much memory. Check large arrays, recursion depth, and accidental unbounded containers.'
    case 'OLE':
      return '### Output Limit Exceeded\n\nYour program printed too much output. Check debug prints and loops that keep writing after the answer is complete.'
    default:
      return [
        `### ${status}`,
        '',
        'The submission did not pass. Compare your program against the statement, sample cases, and edge conditions before looking for implementation details.',
        '',
        message ? `Judge message:\n\n\`\`\`text\n${message.trim()}\n\`\`\`` : ''
      ].join('\n')
  }
}
