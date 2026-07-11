export function submissionDraftKey(user: string, problem: number, language: string, assignment?: number, contest?: number) {
  return ['doj.submission-draft', user, problem, assignment ?? '', contest ?? '', language].join(':')
}
