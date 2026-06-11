import { getRuntimeSettings } from './settings'

export interface CoachingInput {
  status: string
  message: string
  languageId: string
  sourceCode: string
}

export interface CoachingOutput {
  summary: string
  hints: string[]
  nextSteps: string[]
}

export async function createCoachingResponse(input: CoachingInput): Promise<CoachingOutput> {
  const settings = await getRuntimeSettings()
  if (settings.ai.enabled && settings.ai._apiKey)
    return createOpenAiCoachingResponse(input, settings.ai)

  return {
    ...createLocalCoachingResponse(input.status, input.message)
  }
}

async function createOpenAiCoachingResponse(
  input: CoachingInput,
  settings: { _apiKey: string; _baseUrl: string; _model: string }
): Promise<CoachingOutput> {
  if (!settings._apiKey) throw new Error('AI API key is required when AI is enabled')

  const baseUrl = (settings._baseUrl || 'https://api.openai.com/v1').replace(/\/+$/, '')
  const response = await fetch(`${baseUrl}/responses`, {
    method: 'POST',
    headers: {
      'content-type': 'application/json',
      authorization: `Bearer ${settings._apiKey}`
    },
    body: JSON.stringify({
      model: settings._model || 'gpt-5-mini',
      reasoning: { effort: 'low' },
      input: [
        {
          role: 'developer',
          content: [
            'You are an online judge coach.',
            'Help the student debug a non-AC submission without revealing hidden tests, official solutions, or full corrected code.',
            'Return strict JSON only: {"summary": string, "hints": string[], "nextSteps": string[]}.'
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

  return parseCoachingJson(body.output_text ?? readOutputText(body) ?? '')
}

function readOutputText(body: { output?: Array<{ content?: Array<{ text?: string }> }> }) {
  return body.output
    ?.flatMap((item) => item.content ?? [])
    .map((item) => item.text)
    .filter((text): text is string => !!text)
    .join('\n')
}

function parseCoachingJson(text: string): CoachingOutput {
  try {
    const parsed = JSON.parse(text) as Partial<CoachingOutput>
    return {
      summary: String(parsed.summary ?? '评测未通过，请先定位失败类型和关键报错。'),
      hints: Array.isArray(parsed.hints) ? parsed.hints.map(String).slice(0, 5) : [],
      nextSteps: Array.isArray(parsed.nextSteps) ? parsed.nextSteps.map(String).slice(0, 5) : []
    }
  } catch {
    return {
      summary: text.trim() || '评测未通过，请先定位失败类型和关键报错。',
      hints: [],
      nextSteps: ['结合题面、样例和边界条件复查当前实现。']
    }
  }
}

function createLocalCoachingResponse(status: string, message: string): CoachingOutput {
  const trimmedMessage = message.trim()
  switch (status) {
    case 'CE':
      return {
        summary: '编译失败，优先阅读第一条编译错误并定位对应行。',
        hints: ['检查语法、缺失的头文件或导入、语言版本差异。', trimmedMessage ? `编译器输出：${trimmedMessage.slice(0, 300)}` : '没有额外编译输出。'],
        nextSteps: ['本地用同一语言版本编译。', '先修复第一条错误，再重新提交。']
      }
    case 'RE':
      return {
        summary: '运行时错误通常来自越界、除零、空输入假设或递归/栈问题。',
        hints: ['重点检查数组下标、输入解析、边界规模和异常退出路径。', trimmedMessage ? `运行输出：${trimmedMessage.slice(0, 300)}` : '没有额外运行输出。'],
        nextSteps: ['用最小样例和边界样例复现。', '为关键变量添加本地断言或日志后再移除。']
      }
    case 'TLE':
      return {
        summary: '程序超时，当前算法复杂度或循环终止条件可能不满足限制。',
        hints: ['重新估算最坏情况复杂度。', '检查是否存在无法收敛的循环或重复计算。'],
        nextSteps: ['用最大规模数据做本地压测。', '考虑预处理、剪枝或更高效的数据结构。']
      }
    case 'MLE':
      return {
        summary: '内存超限，数据结构规模或递归深度可能超过限制。',
        hints: ['检查大数组维度、缓存容器和递归栈。', '确认是否保留了不必要的中间结果。'],
        nextSteps: ['按最大输入估算内存。', '尝试滚动数组或流式处理。']
      }
    case 'OLE':
      return {
        summary: '输出超限，通常是调试输出未删除或循环持续打印。',
        hints: ['检查所有调试 print。', '确认输出循环能正确停止。'],
        nextSteps: ['只保留题目要求的输出。', '用小样例核对输出行数和格式。']
      }
    default:
      return {
        summary: `${status}：提交未通过，请从题意、样例和边界条件开始排查。`,
        hints: [trimmedMessage ? `评测消息：${trimmedMessage.slice(0, 300)}` : '没有额外评测消息。', '重点对比输出格式、边界输入和特殊分支。'],
        nextSteps: ['手工构造极小、极大和特殊样例。', '逐步核对核心状态转移或判断逻辑。']
      }
  }
}
