# API 与权限设计

## 总体原则

- API 返回 shape 要稳定、简洁、领域化。
- 服务端裁剪敏感字段；前端隐藏不是安全边界。
- 管理权限统一使用 `users.admin`。
- 后端实现应通过中间件简化，例如 `requireAuth`、`requireAdmin`、`optionalUser`、`canViewSubmission`。
- 所有列表分页参数固定为 `page` 和 `pageSize`，默认 `page=1&pageSize=50`，`pageSize` 上限固定 100，返回 `items`、`page`、`pageSize`、`total`。
- 排序参数固定为 `sort` 和 `order`；`order` 只允许 `asc` 或 `desc`，接口未声明支持排序时忽略这两个参数。
- 所有时间戳字段使用 ISO 8601 UTC 字符串；heatmap 例外，API 接收 `tz` 参数并按该 IANA 时区聚合成本地日期桶。
- 错误响应避免泄露内部信息；管理员接口可返回更具体配置错误。

统一错误响应：

```ts
interface ApiError {
  error: {
    code: string
    message: string
    issues?: { path: string; message: string }[]
  }
}
```

状态码规则：

- `400`：请求格式或业务校验错误。
- `401`：未登录、session 失效或 token 无效。
- `403`：已登录但无权限或账号禁用。
- `404`：资源不存在，或普通用户访问隐藏/软删除资源。
- `409`：唯一约束、删除受限、状态冲突。
- `413`：body、源码或上传文件超过上限。
- `429`：限流。
- `500`：未预期服务端错误；普通用户只返回通用 message。
- Zod 校验错误固定返回 `400`，`code='VALIDATION_ERROR'`，`issues` 填字段路径和错误消息。
- `429` 必须带 `Retry-After` header，单位为秒，错误 code 固定 `RATE_LIMITED`。
- 资产路径校验错误固定返回 `400`，`code='INVALID_ASSET_PATH'`。

## Auth

接口：

- `POST /api/auth/register`
- `POST /api/auth/login`
- `POST /api/auth/logout`
- `GET /api/auth/self`
- `PATCH /api/auth/self`
- `POST /api/auth/email-code`
- `PATCH /api/auth/email`

规则：

- 注册默认关闭。
- 开放注册要求 SMTP 已配置并启用验证码。
- 不使用邀请码。
- 注册验证码发送到注册邮箱；修改邮箱验证码发送到新邮箱。
- `POST /api/auth/email-code` 请求体固定为 `{ purpose: 'register' | 'change-email', email: string }`；`change-email` 需要登录。
- `POST /api/auth/register` 请求体固定为 `{ name: string, email: string, password: string, code: string }`，必须提交邮箱验证码。
- 注册或修改邮箱遇到 email 已存在时固定返回 409，`code='EMAIL_EXISTS'`。
- `POST /api/auth/login` 请求体固定为 `{ user: string, password: string }`，`user` 可为用户名或邮箱。
- `PATCH /api/auth/self` 请求体固定为 `{ introduction?: string, currentPassword?: string, password?: string }`；修改密码时必须提交正确 `currentPassword`。
- `PATCH /api/auth/email` 请求体固定为 `{ email: string, code: string }`，必须提交发往新邮箱的验证码。
- `POST /api/auth/register` 和 `POST /api/auth/login` 响应固定为 `{ token: string, user: AuthSelf }`。
- `GET /api/auth/self` 响应固定为 `AuthSelf`。
- 未登录请求 `GET /api/auth/self` 固定返回 401，`code='UNAUTHORIZED'`。
- `AuthSelf` 固定为 `{ id: number, name: string, email: string, introduction: string, admin: boolean, disabled: boolean, mustChangePassword: boolean, avatarUrl: string }`。
- `mustChangePassword=true` 时，除 `GET /api/auth/self` 和用于修改密码的 `PATCH /api/auth/self` 外，其他登录态接口固定返回 403。
- 禁用用户登录固定返回 403；禁用用户的既有 session 在任意鉴权接口上立即失效并返回 401。
- 密码哈希、session、限流、验证码、body size 上限以 `02-data-and-storage.md` 的固定安全参数为准。
- Gravatar 头像由 API 返回 `avatarUrl`；email 先 trim + lowercase，再用 MD5 hex 生成 hash，URL 固定为 `https://www.gravatar.com/avatar/{hash}?d=identicon&s=80`。
- 邮件发送失败时 `POST /api/auth/email-code` 不写入 Redis，固定返回 500，`code='SMTP_SEND_FAILED'`。

## Public Config

接口：

- `GET /api/config`

返回 shape 固定：

```ts
interface PublicConfig {
  signup: boolean
  guestAccess: boolean
  publicCode: boolean
  smtpConfigured: boolean
  aiEnabled: boolean
  notice: string
}
```

说明：

- `publicCode` 只作为提交表单 `public` 勾选框初始值，不强制覆盖用户选择。
- 不返回 `_` 私有字段原值。

## Admin Settings

接口：

- `GET /api/admin/settings`
- `PATCH /api/admin/settings`

分区：

- `general`
- `smtp`
- `ai`

规则：

- `_` 私有字段只在写入时接收；读取时返回 `xxxSet` 或空占位。
- 开启注册时校验 smtp 可用性或至少校验 smtp enabled/config 完整。

## Users and Groups

接口：

- `GET /api/users/:id`
- `GET /api/admin/users`
- `POST /api/admin/users`
- `GET /api/admin/users/:id`
- `PATCH /api/admin/users/:id`
- `POST /api/admin/users/:id/reset-password`
- `GET /api/admin/groups`
- `POST /api/admin/groups`
- `PATCH /api/admin/groups/:id`
- `DELETE /api/admin/groups/:id`
- `GET /api/admin/groups/:id/users`
- `POST /api/admin/groups/:id/users`
- `DELETE /api/admin/groups/:id/users/:userId`

规则：

- `GET /api/users/:id` 是公开个人主页接口，返回 `{ id, name, introduction, avatarUrl, solved, submissions, createdAt }`，不返回 email；用户本人和管理员通过 `GET /api/auth/self` 或 admin users 接口查看 email。
- Users 支持改用户名、邮箱、简介、admin、disabled。
- 不允许取消最后一个管理员的 admin，不允许禁用最后一个未禁用管理员；违反时返回 409，`code='LAST_ADMIN'`。
- 管理员可以禁用自己，但如果自己是最后一个未禁用管理员则返回 `LAST_ADMIN`。
- `POST /api/admin/users` 请求体固定为 `{ name: string, email: string, password: string, admin?: boolean, disabled?: boolean }`。
- `GET /api/admin/users/:id` 响应固定为 `{ id, name, email, introduction, admin, disabled, mustChangePassword, createdAt, updatedAt }`。
- `PATCH /api/admin/users/:id` 请求体固定为 `{ name?: string, email?: string, introduction?: string, admin?: boolean, disabled?: boolean, mustChangePassword?: boolean }`。
- `POST /api/admin/users/:id/reset-password` 请求体固定为 `{ password?: string }`；传入 password 时使用该密码，未传时 server 生成 12 字节随机数的 base64url 字符串作为新密码；响应固定返回 `{ password: string }`，只返回这一次。
- `POST /api/admin/groups` 请求体固定为 `{ name: string }`。
- `PATCH /api/admin/groups/:id` 请求体固定为 `{ name: string }`。
- `POST /api/admin/groups/:id/users` 请求体固定为 `{ userId: number }`。
- Groups 添加成员使用弹框搜索，不用外显长 select。
- 不做 group manager。

## Problems and Assets

题目接口：

- `GET /api/problems`
- `GET /api/problems/:id`
- `POST /api/admin/problems`
- `PATCH /api/admin/problems/:id`
- `DELETE /api/admin/problems/:id`
- `POST /api/admin/problems/:id/restore`

资产接口：

- `GET /api/admin/problems/:id/assets`
- `GET /api/admin/problems/:id/assets/content?path=`
- `PUT /api/admin/problems/:id/assets/content`
- `POST /api/admin/problems/:id/assets/upload`
- `DELETE /api/admin/problems/:id/assets?path=`
- `POST /api/admin/problems/:id/assets/pe-checker`
- `GET /api/problems/:id/assets/:filename`

说明：

- 管理端 list/read/write/delete 整个资产树。
- 普通用户只能 GET 公开题目的 `assets/{filename}`，不能 list。
- 公开读取接口只服务 `assets/` 平铺文件，不读 `data/` 或根目录评测资源。
- `assets/` 不允许子目录。
- `pe-checker` 按钮写入预设 `Dockerfile` 和 `checker.cc`，并说明 PE 是 ICPC 风格。
- `GET /api/problems` 对普通用户和访客只返回 `visible=true` 且 `deletedAt IS NULL` 的题目；管理员列表可包含隐藏题和软删除题。
- `GET /api/problems/:id` 对普通用户和访客访问 hidden/deleted/missing 题目都返回 404，不暴露存在性；管理员可读取 hidden/deleted 题目。
- `POST /api/submissions` 只允许提交 `visible=true` 且 `deletedAt IS NULL` 的题目，管理员也遵守同一规则。
- `GET /api/problems/:id/assets/:filename` 只允许读取 `visible=true` 且 `deletedAt IS NULL` 题目的 `assets/` 文件。
- `GET /api/problems` 查询参数固定支持 `page`、`pageSize`、`q`、`tag`；`q` 匹配题号、标题和标签，大小写不敏感。
- `GET /api/problems` 默认按 `id ASC` 排序；首版不开放其他排序。
- `q` 匹配规则固定为：纯数字时同时匹配 `problem.id == Number(q)` 和标题子串；非纯数字时匹配标题子串和 tag 完全相等；标题匹配大小写不敏感，tag 匹配大小写敏感。
- `tag` 参数固定为单个 tag 精确匹配，大小写敏感；首版不支持多 tag 同时筛选。
- `POST /api/admin/problems` 请求体固定为 `{ title: string, content?: string, timeLimitMs?: number, memoryLimitBytes?: number, tags?: string[], visible?: boolean }`。
- `PATCH /api/admin/problems/:id` 请求体固定为 `{ title?: string, content?: string, timeLimitMs?: number, memoryLimitBytes?: number, tags?: string[], visible?: boolean }`。
- 新建默认值固定为 `content=''`、`timeLimitMs=1000`、`memoryLimitBytes=134217728`、`tags=[]`、`visible=false`。
- `timeLimitMs` 合法范围固定 `100..60000`；`memoryLimitBytes` 合法范围固定 `16777216..1073741824`；tags 最多 10 个，每个 tag 长度 1-32。

题目详情 shape 固定：

```ts
interface ProblemListItem {
  id: number
  title: string
  tags: string[]
  passRate: number
  solvedUsers: number
  attemptedUsers: number
  solved: boolean
  visible: boolean
}

interface ProblemDetail {
  id: number
  title: string
  content: string
  timeLimitMs: number
  memoryLimitBytes: number
  tags: string[]
  passRate: number
  solvedUsers: number
  attemptedUsers: number
  solved: boolean
  recentSubmission: SubmissionListItem | null
  visible: boolean
  deletedAt: string | null
  createdAt: string
  updatedAt: string
}
```

- `GET /api/problems` 返回分页 `ProblemListItem`。
- `passRate` 固定为 `solvedUsers / attemptedUsers`，`attemptedUsers=0` 时为 0；API 返回 0 到 1 的 number，前端展示百分比时保留 1 位小数。
- 未登录用户的 `solved=false`、`recentSubmission=null`。
- `ProblemDetail.recentSubmission` 必须使用与 submissions 列表相同的权限和封榜裁剪口径。

资产 API shape 固定：

```ts
type ProblemAssetSection = 'data' | 'assets' | 'root'

interface ProblemAssetItem {
  path: string
  name: string
  section: ProblemAssetSection
  size: number
  contentType: string
  updatedAt: string
  text: boolean
}

interface ProblemAssetContent {
  path: string
  contentType: string
  text: boolean
  content: string
  encoding: 'utf8' | 'base64'
}

interface ProblemAssetUploadResult {
  path: string
  url: string | null
}
```

- `GET /api/admin/problems/:id/assets` 返回 `ProblemAssetItem[]`，按 `section` 的 `data, assets, root` 顺序再按 path 升序。
- `GET /api/admin/problems/:id/assets/content?path=` 对 UTF-8 文本返回 `encoding='utf8'` 和原文；二进制返回 `encoding='base64'`。
- `PUT /api/admin/problems/:id/assets/content` 请求体固定为 `{ path: string, content: string, encoding: 'utf8' | 'base64', contentType?: string }`，返回 `ProblemAssetItem`。
- `POST /api/admin/problems/:id/assets/upload` 使用 multipart 字段 `file` 和 `path`，返回 `ProblemAssetUploadResult`；上传到 `assets/{filename}` 时 `url` 固定为 `/api/problems/{id}/assets/{filename}`，供 `md-editor-v3` 图片回填。
- `DELETE /api/admin/problems/:id/assets?path=` 返回 `{ ok: true }`。
- `POST /api/admin/problems/:id/assets/pe-checker` 返回写入后的 `ProblemAssetItem[]`。

题目创建流程：

- 第一步创建基础信息，`content` 可为空，拿到 ID。
- 第二步编辑题面与附件。
- 第三步上传 data 和评测资源。

## Submissions

接口：

- `GET /api/submissions`
- `GET /api/submissions/:id`
- `POST /api/submissions`
- `POST /api/submissions/:id/coach`

创建：

- 前端只提交 `problemId`、`languageId`、`code`、`public`。
- 可传当前页面上下文，但最终 `contestId`/`assignmentId` 由后端判断。
- 同一题目同时落在多个进行中比赛时，若请求显式传入 `contestId` 且用户有权参与该比赛，则使用该 `contestId`；否则选择 `startAt` 最晚、再按 `id` 最大的进行中比赛。
- 作业无开始时间；若请求显式传入 `assignmentId` 且用户在该作业范围内、题目属于该作业、`createdAt < endAt`，则使用该 `assignmentId`；否则选择 `endAt` 最早、再按 `id` 最小的未截止作业。
- 比赛窗口固定为 `[startAt, endAt)`；作业截止固定为 `createdAt < endAt`。
- 后端写入 WAITING submission 并入队。
- 首版不提供 rejudge 接口；不存在 `POST /api/admin/submissions/:id/rejudge`。

列表：

- 默认分页 50。
- 查询参数固定支持 `page`、`pageSize`、`status`、`problemId`、`userId`、`languageId`、`contestId`、`assignmentId`。
- 敏感字段裁剪。

详情：

- 源码只对本人、管理员或 public 可见。
- 比赛封榜、隐藏题、作业范围等都在服务端裁剪。
- live progress 通过 WS，HTTP detail 返回最终或当前快照。

提交列表和详情 shape 固定：

```ts
type SubmissionDisplayStatus = JudgeStatus | 'SUBMITTED' | 'JUDGED'

interface SubmissionListItem {
  id: number
  problem: { id: number; title: string } | null
  user: { id: number; name: string; avatarUrl: string }
  languageId: string
  status: JudgeStatus | null
  displayStatus: SubmissionDisplayStatus
  score: number | null
  timeMs: number | null
  memoryBytes: number | null
  public: boolean
  contestId: number | null
  assignmentId: number | null
  createdAt: string
  updatedAt: string
}

interface SubmissionCaseView {
  caseNo: number
  status: JudgeStatus
  timeMs: number
  memoryBytes: number
  score: number
  message: string | null
}

interface SubmissionDetail extends SubmissionListItem {
  code: string | null
  message: string | null
  cases: SubmissionCaseView[]
  canCoach: boolean
}
```

裁剪规则：

- 无权查看源码时 `code=null`。
- 无权查看题目或题目已隐藏/删除时，普通用户看到 `problem=null`；管理员仍看到题目摘要。
- 非封榜、非隐藏裁剪场景下，`displayStatus` 固定等于真实 `status`。
- 比赛封榜裁剪后，真实 `status`、`score`、`timeMs`、`memoryBytes`、`message` 和 `cases` 按比赛规则隐藏或置空；列表也必须使用同一裁剪口径。
- OI 封榜期普通用户的 `displayStatus` 固定映射为：原始 `WAITING` 显示 `SUBMITTED`，原始 `JUDGING` 显示 `JUDGING`，其他已完成状态显示 `JUDGED`；此时 `status=null`、`score/timeMs/memoryBytes=null`、`message=null`、`cases=[]`。

AI coaching：

- 不落长期表。
- Redis 缓存 TTL 固定 10 分钟。
- 由用户主动点击触发，只允许在 `settings.ai.enabled=true` 时触发；只允许提交本人或管理员触发；普通用户不能对他人提交触发 coach，即使源码 public。
- `canCoach=true` 的条件固定为：AI enabled，submission 已完成，当前用户是提交本人或管理员，且用户有权查看该提交详情。
- 输出语言固定跟随用户界面语言；首版界面语言固定中文时，coach 输出中文。
- 输入 prompt 固定包含题面摘要、语言、源码、最终 status、score、message 和最多前 20 个非 AC case 摘要。
- 题面摘要固定取 Markdown 纯文本前 1000 字符；源码最多取前 12000 字符；每个非 AC case 摘要固定为 `caseNo/status/message`，message 截到 300 字符。
- 输出结构固定为 `{ summary: string, hints: string[], nextSteps: string[] }`。

## WebSocket

接口：

- `GET /api/ws`
- Agent 连接接口固定为 `GET /api/agents/connect`。

浏览器消息：

```ts
type BrowserToServer =
  | { type: 'subscribe-submission'; submissionId: number }
  | { type: 'unsubscribe-submission'; submissionId: number }
  | { type: 'subscribe-feed'; scope: 'self' | 'page'; submissionIds?: number[] }
  | { type: 'unsubscribe-feed'; scope: 'self' | 'page' }
```

server 推送：

```ts
type ServerToBrowser =
  | { type: 'submission-progress'; submissionId: number; progress: JudgeProgress }
  | { type: 'submission-result'; submissionId: number; result: SubmissionDetail }
  | { type: 'submission-feed'; item: SubmissionListItem }
```

权限：

- 浏览器 WS 使用登录 token 鉴权，固定读取 `Sec-WebSocket-Protocol: doj-auth.<token>`；不使用 query string token。
- Agent WS 使用 `Authorization: Bearer <SECRET>`。
- 订阅时校验是否有权查看 submission。
- `subscribe-feed` 的 `self` 订阅推送当前用户新建或更新的提交；`page` 订阅只推送 `submissionIds` 中有权限查看的提交更新，用于提交列表当前页。
- 断线重连后先 HTTP 拉详情。
- 浏览器 WS 由 server 每 30 秒发送 `{ type: 'ping' }`；浏览器收到后 10 秒内发送 `{ type: 'pong' }`，server 60 秒未收到 pong 即关闭连接。

## Ranking And Home Data

接口：

- `GET /api/ranking`
- `GET /api/tags`
- `GET /api/home/heatmap`
- `GET /api/home/recommended-problems`

规则：

- 排行榜默认分页 50，按 AC 数降序、提交数升序、最近 AC 时间升序、userId 升序。
- `GET /api/ranking` 查询参数固定支持 `page`、`pageSize`。
- 全局排行榜只包含至少有一次提交的用户，包含 0 AC 用户；`total` 是至少有一次提交的用户数，不包含从未提交的用户。
- `GET /api/tags` 返回公开且未删除题目的去重 tags 和题目数量。
- heatmap 查询参数固定支持 `tz`，值为 IANA timezone，例如 `Asia/Shanghai`；server 按该时区聚合滚动 365 天提交次数并返回 `{ date: 'YYYY-MM-DD', count: number }[]`，包含该时区今天和向前 364 天。未传或非法时使用 `UTC`；访客固定返回 401。
- 推荐题返回公开、未删除、用户未 AC 的题目，按题号升序稳定排序，登录用户和访客都固定最多 10 题；访客返回公开、未删除题目中题号最小的前 10 题。
- 通过率固定为 solved users / attempted users；attempted users 是至少提交过一次该题的去重用户数，包含 CE/SE；attempted users 为 0 时返回 `0`。

返回 shape 固定：

```ts
interface RankingRow {
  rank: number
  user: { id: number; name: string; avatarUrl: string }
  solved: number
  submissions: number
  lastAcAt: string | null
}

interface TagItem {
  name: string
  count: number
}
```

- `GET /api/ranking` 返回分页 `RankingRow`。
- `GET /api/tags` 返回 `TagItem[]`，按 `count DESC, name ASC` 排序。
- `GET /api/home/recommended-problems` 返回 `ProblemListItem[]`。

## Assignments

接口：

- `GET /api/assignments`
- `POST /api/admin/assignments`
- `GET /api/admin/assignments/:id`
- `PATCH /api/admin/assignments/:id`
- `DELETE /api/admin/assignments/:id`
- `GET /api/admin/assignments/:id/report`
- `GET /api/my/assignments`
- `GET /api/my/assignments/:id`

规则：

- 管理列表分页 50。
- 我的作业分页 50。
- 列表查询参数固定支持 `page`、`pageSize`、`status`，其中 `status` 为 `current` 或 `past`；作业没有 upcoming 状态。
- 创建/编辑支持 groups + individual users + problems。
- `POST /api/admin/assignments` 请求体固定为 `{ title: string, description?: string, endAt: string, groupIds?: number[], userIds?: number[], problemIds: number[] }`。
- `PATCH /api/admin/assignments/:id` 请求体同创建字段但全部可选；截止前可编辑全部字段，截止后只允许 `title`、`description`。
- 创建/编辑只能选择 `visible=true` 且 `deletedAt IS NULL` 的题目；若已关联题目后来隐藏或软删除，普通用户详情显示不可用且不能提交，报告继续保留历史提交统计。
- 作业创建后立即生效；没有未开始状态。
- 删除为软删除。
- 报告 attempts 固定统计提交时写入该 `assignmentId` 的所有提交，包含 CE/SE，不按用户去重。

我的作业与报告 shape 固定：

```ts
interface AssignmentProblemProgress {
  problemId: number
  title: string | null
  unavailable: boolean
  attempts: number
  bestScore: number
  ac: boolean
  lastSubmissionId: number | null
  lastSubmittedAt: string | null
}

interface MyAssignmentDetail {
  id: number
  title: string
  description: string
  endAt: string
  problems: AssignmentProblemProgress[]
}

interface AssignmentReportRow {
  user: { id: number; name: string; avatarUrl: string }
  problems: Record<number, AssignmentProblemProgress>
}
```

## Contests

接口：

- `GET /api/contests`
- `GET /api/contests/:id`
- `POST /api/admin/contests`
- `PATCH /api/admin/contests/:id`
- `DELETE /api/admin/contests/:id`
- `GET /api/contests/:id/scoreboard`
- `GET /api/admin/contests/:id/scoreboard/full`

说明：

- `full scoreboard` 是管理员查看真实榜单的接口，UI 文案用“查看真实榜单”，不要用 reveal。
- 普通 scoreboard 对所有普通用户一视同仁，不做个性化榜单。
- `POST /api/admin/contests` 请求体固定为 `{ title: string, description?: string, type: 'OI' | 'ICPC', startAt: string, endAt: string, freezeAt?: string | null, problemIds: number[] }`。
- `PATCH /api/admin/contests/:id` 请求体同创建字段但全部可选；进行中只允许 `title`、`description`。
- 比赛题号 key 由 `problemIds` 顺序自动生成，固定为 Excel 列名规则：`A..Z, AA, AB, ...`，首版最多 100 题，超过返回 400。
- OI 榜单每题取该用户在本场比赛上下文中最后一次提交的 score；最后一次按 `createdAt DESC, id DESC` 判定，封榜只裁剪普通用户展示，不改变真实统计口径。
- OI 比赛 `endAt` 前，普通 scoreboard 不展示真实 per-problem score、总分和排名，固定展示每题已提交/未提交占位；`endAt` 后展示真实榜单。
- OI full scoreboard 和结束后的 public scoreboard 按 `totalScore DESC, lastEffectiveSubmitAt ASC, userId ASC` 排序；`lastEffectiveSubmitAt` 是该用户所有有提交的比赛题目中，被 OI 最后提交规则选中的 submission.createdAt 最大值，没有任何提交时为 `null` 且排序放最后；rank 使用竞赛排名，并列判定只看 `totalScore` 和 `lastEffectiveSubmitAt`，不看 userId，例如 `1,1,3`。
- ICPC 榜单按 solved 降序、penalty 升序、userId 升序；罚时为首次 AC 分钟数向下取整 + AC 前计罚错误次数 * 20，计罚错误包含 WA、PE、TLE、MLE、OLE、RE，不包含 CE 和 SE。
- ICPC rank 使用竞赛排名，并列判定只看 `solved` 和 `penalty`，不看 userId。
- OI 比赛结束前，普通用户 submission detail 裁剪 status、score、case、message。
- ICPC 封榜后，普通用户仍可在自己的 submission detail 看到真实 verdict；榜单上 freeze 后提交显示 pending。
- ICPC public scoreboard 在 `endAt` 后自动解封，展示真实榜单；无需管理员手动操作。
- ICPC `freezeAt` 按数据库保存值执行；管理端 UI 默认预填 `endAt - 1h`，保存为空表示不封榜。OI `freezeAt` 固定为 `NULL`。
- scoreboard `frozen` 判定固定为：OI 在 `startAt <= now < endAt` 为 true；ICPC 在 `freezeAt != null && freezeAt <= now < endAt` 为 true；其他情况为 false。
- 首版无报名、私有比赛和参赛范围；任何登录用户在比赛窗口内提交比赛题目都会进入该比赛榜单。
- 创建/编辑只能选择 `visible=true` 且 `deletedAt IS NULL` 的题目；若已关联题目后来隐藏或软删除，普通用户详情显示不可用且不能继续提交，scoreboard 继续保留历史提交统计。
- 列表分页 50，支持当前/即将/历史筛选。
- 列表查询参数固定支持 `page`、`pageSize`、`status`，其中 `status` 为 `current`、`upcoming` 或 `past`。
- scoreboard 查询参数固定支持 `page`、`pageSize`，默认 50，上限 100；`total` 返回总 rows 数。
- 未开始普通用户不看题目列表。

scoreboard 返回 shape 固定：

```ts
type ScoreboardMode = 'public' | 'full'

interface ContestScoreboard {
  contest: {
    id: number
    type: 'OI' | 'ICPC'
    startAt: string
    endAt: string
    freezeAt: string | null
    frozen: boolean
    mode: ScoreboardMode
  }
  problems: { problemId: number; key: string; title: string; sort: number }[]
  rows: ScoreboardRow[]
  page: number
  pageSize: number
  total: number
  generatedAt: string
}

interface ScoreboardRow {
  user: { id: number; name: string; avatarUrl: string }
  rank: number | null
  totalScore?: number
  lastEffectiveSubmitAt?: string | null
  solved?: number
  penalty?: number
  problems: Record<string, OIProblemCell | ICPCProblemCell>
}

interface OIProblemCell {
  submitted: boolean
  pending: boolean
  score: number | null
  status: JudgeStatus | null
  submissionId: number | null
  submittedAt: string | null
}

interface ICPCProblemCell {
  submitted: boolean
  pending: boolean
  accepted: boolean
  attempts: number
  wrongAttempts: number
  acceptedAt: string | null
  penalty: number
  status: JudgeStatus | null
  submissionId: number | null
}
```

scoreboard 裁剪规则：

- `problems` 的 key 使用比赛题号，例如 `A`、`B`；`rows[].problems` 以该 key 作为对象键。
- OI public scoreboard 在 `endAt` 前固定 `rank=null`、`totalScore` 不返回、每个 cell 的 `score/status/submissionId/submittedAt` 为 `null`，只保留 `submitted` 和 `pending`。
- OI full scoreboard 和 OI 结束后的 public scoreboard 返回 `rank`、`totalScore` 和真实 cell。
- OI 最后一次提交若仍为 `WAITING` 或 `JUDGING`，该题 cell 固定 `pending=true`、`score=0`、`status` 为当前运行态，`totalScore` 先按 0 累加；评测完成后重新计算 scoreboard。
- ICPC public scoreboard 在 `freezeAt` 后对 freeze 后提交只更新 `submitted=true`、`pending=true`，不更新 `accepted/attempts/wrongAttempts/acceptedAt/penalty/status/submissionId`。
- ICPC `attempts` 固定统计首次 AC 前的所有已公开提交，包含 CE 和 SE，不包含仍隐藏的 pending；首次 AC 之后的提交不计入 `attempts`。
- ICPC `wrongAttempts` 固定统计首次 AC 前的计罚错误 `WA/PE/TLE/MLE/OLE/RE`，不含 CE/SE，不含 pending，不含 AC 后提交。
- ICPC full scoreboard 始终返回真实榜单。
- `pending` 表示该 cell 存在未公开结果的提交，包含 OI 封榜期已评测但隐藏结果、ICPC freeze 后提交、以及仍在 WAITING/JUDGING 的提交。

比赛详情 shape 固定：

```ts
interface ContestDetail {
  id: number
  title: string
  description: string
  type: 'OI' | 'ICPC'
  startAt: string
  endAt: string
  freezeAt: string | null
  problems: { problemId: number; key: string; title: string | null; unavailable: boolean; sort: number }[]
}
```

- 未开始时普通用户 `ContestDetail.problems=[]`；管理员仍可看到完整 problems。
- 未开始时普通用户请求 scoreboard 固定返回 403，`code='CONTEST_NOT_STARTED'`；管理员 full scoreboard 可访问。

## Discussion

接口：

- `GET /api/discussion/topics`
- `POST /api/discussion/topics`
- `GET /api/discussion/topics/:id`
- `POST /api/discussion/topics/:id/posts`

规则：

- topic 列表固定按 `pinned DESC, updatedAt DESC, id DESC` 排序。
- `GET /api/discussion/topics` 查询参数固定支持 `page`、`pageSize`。
- `POST /api/discussion/topics` 请求体固定为 `{ title: string, content: string, tags?: string[] }`，创建 topic 时事务性创建首楼 post。
- `POST /api/discussion/topics/:id/posts` 请求体固定为 `{ content: string }`。
- topic title 长度固定 1-100；topic/post content 长度固定 1-20000；tags 最多 5 个，每个 tag 长度 1-32。
- 创建回复时必须回写 `topics.updatedAt=now`。

首版可不做：

- 编辑。
- 删除。
- 嵌套回复。
- 站内图片上传。

如果做删除：首楼删除等价 topic 软删除。

讨论 shape 固定：

```ts
interface TopicListItem {
  id: number
  title: string
  tags: string[]
  pinned: boolean
  author: { id: number; name: string; avatarUrl: string }
  updatedAt: string
  createdAt: string
}

interface PostView {
  id: number
  topicId: number
  user: { id: number; name: string; avatarUrl: string }
  content: string
  createdAt: string
}

interface TopicDetail extends TopicListItem {
  posts: PostView[]
}
```

## Languages

接口：

- `GET /api/languages`
- `GET /api/admin/languages`
- `POST /api/admin/languages`
- `PATCH /api/admin/languages/:id`
- `DELETE /api/admin/languages/:id`

规则：

- `GET /api/languages` 是提交表单使用的公开接口，返回 `LanguagePublic[]`，按 `sort ASC, id ASC` 排序，不返回 Dockerfile。
- `GET /api/admin/languages` 返回 `LanguageAdmin[]`，按 `sort ASC, id ASC` 排序。
- seed 只有 `cpp/main.cc`。
- 允许多语言。
- 自定义语言 `id` 固定为 1-32 个字符，只允许小写 ASCII 字母、数字和连字符，且首字符必须是小写字母；重复 ID 返回 409。
- `source` 文件名固定为 1-64 个字符，只允许 ASCII 字母、数字、下划线、点和连字符，禁止 `/`、`\`、空格和 `..`；同一语言表内不要求唯一。
- 不允许删除最后一个语言，返回 409 `code='LAST_LANGUAGE'`；已被历史 submissions 引用的语言删除固定返回 409 `code='LANGUAGE_IN_USE'`；重复语言 ID 返回 409 `code='LANGUAGE_EXISTS'`。
- `POST /api/admin/languages` 请求体固定为 `{ id: string, name: string, source: string, dockerfile: string, sort?: number }`；`sort` 默认 0。
- `PATCH /api/admin/languages/:id` 请求体固定为 `{ name?: string, source?: string, dockerfile?: string, sort?: number }`。
- Dockerfile 在语言页文本编辑。
- 语言 Dockerfile 必须在 build 阶段编译用户源码，并通过 `CMD` 或 `ENTRYPOINT` 定义运行入口。
- 用户源码以 `source` 文件名放在 build context 根目录；B 镜像 build 失败即 CE。

语言 shape 固定：

```ts
interface LanguagePublic {
  id: string
  name: string
  source: string
  sort: number
}

interface LanguageAdmin extends LanguagePublic {
  dockerfile: string
  createdAt: string
  updatedAt: string
}
```

## Agents

Agent 内部包含 Runner 执行模块，但 API 只暴露 Agent 运行态，不暴露 Runner 作为独立服务。


接口：

- `GET /api/admin/agents`
- `GET /api/admin/agents/instructions`

规则：

- 只展示运行态 Agent 和接入指引。
- 不 CRUD PG agent。
- 不 labels。
- concurrency 只读，由 Agent 启动参数上报。
- secret 来自 env，不做在线刷新。

Agent shape 固定：

```ts
interface AgentRuntimeView {
  key: string
  name: string
  concurrency: number
  activeJobs: number
  version: string
  connectedAt: string
  lastHeartbeat: string
}
```

`GET /api/admin/agents/instructions` 固定返回 `{ server: string, secretEnv: 'SECRET', command: string }`，`command` 是推荐启动命令文本，必须使用 `SERVER`、`SECRET`、`AGENT_NAME`、`AGENT_CONCURRENCY` 环境变量名。

## Guest Access

设置名固定为 `general.guestAccess`。

含义：访客能否访问公开内容，包括题库、题目详情、公开提交、排行、比赛列表/详情和讨论列表。即使开启，也必须遵守隐藏题、软删除、封榜和敏感字段裁剪。
