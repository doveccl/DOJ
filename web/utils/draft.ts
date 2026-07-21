export function submissionDraftKey(user: string, problem: number, language: string, assignment?: number, contest?: number) {
  return ['doj.submission-draft', user, problem, assignment ?? '', contest ?? '', language].join(':')
}

export function discussionDraftKey(user: string, kind: 'new' | 'edit' | 'reply', discussion?: number) {
  return ['doj.discussion-draft', user, kind, discussion ?? ''].join(':')
}
