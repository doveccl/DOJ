import type { JudgeProgress } from '@doj/shared/judge'
import { redisDel, redisGetJson, redisSetJson } from './redis'

const progressTtlSeconds = 10 * 60
const memoryProgress = new Map<number, JudgeProgress>()
let broadcaster: ((submissionId: number, progress: JudgeProgress) => void) | null = null

export function setProgressBroadcaster(
  nextBroadcaster: (submissionId: number, progress: JudgeProgress) => void
) {
  broadcaster = nextBroadcaster
}

export async function saveJudgeProgress(submissionId: number, progress: JudgeProgress) {
  memoryProgress.set(submissionId, progress)
  await redisSetJson(progressKey(submissionId), progress, progressTtlSeconds)
  broadcaster?.(submissionId, progress)
}

export async function getJudgeProgress(submissionId: number) {
  const cached = await redisGetJson<JudgeProgress>(progressKey(submissionId))
  return cached ?? memoryProgress.get(submissionId) ?? null
}

export async function clearJudgeProgress(submissionId: number) {
  memoryProgress.delete(submissionId)
  await redisDel(progressKey(submissionId))
}

function progressKey(submissionId: number) {
  return `progress:submission:${submissionId}`
}
