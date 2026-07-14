# Frontend Rules

- `web/` contains frontend source. The Vite HTML entry stays at repository root as `index.html`.
- OpenAPI-generated TypeScript schema/client wrappers live in `web/client/`.
- API/server state uses TanStack Query. Do not keep a second independent `useState` cache for the same server data.
- Forms and modals should use React local state and antd Form.
- Do not introduce Redux or Zustand unless there is a real global client-state problem.

## UI Rules

- Prefer antd native components and documented props over custom CSS.
- Check `https://ant.design/llms.txt` or the component Markdown docs when using unfamiliar antd APIs.
- Do not use deprecated antd props.
- Do not specify component `size` by default. Regular page Card headers use default-size controls; `size="small"` Cards, including dense Cards inside modals, use `small` controls consistently.
- Keep `.appLayout` at `100vw` with horizontal overflow clipped on the document; this intentionally prevents the centered page from changing width when the vertical scrollbar appears.
- Keep modal geometry stable with `scrollbar-gutter: stable both-edges` on its scroll container; do not replace it with JavaScript width measurement.
- Use Tag for row state when table row styling would require custom CSS.
- When text and Tag share a line, use an explicit center-aligned Flex layout. Remove Tag's default inline-end margin when the parent already supplies a gap.
- Free-form tags from problem/discussion/user data must render through the shared entity tag component so long tags get antd ellipsis and tooltip behavior consistently.
- Problem references in tables, lists, timelines, and cards should use the shared problem link component; set a max width at the call site only when the surrounding layout needs a fixed budget.
- For table overflow, give ellipsized text columns an explicit `width` and use `ellipsis: { showTitle: false }` with `Typography.Text` ellipsis. Do not use `tableLayout="fixed"` or shared CSS as a substitute for column sizing. `TagList` handles its own item width; do not add table column ellipsis just for tag lists.
- Table filter toolbars use `Flex vertical gap={16}` around the toolbar and table, with `tableToolbar` plus `tableToolbarForm` on inline antd forms so wrapped fields keep row spacing.
- Table action columns must have a visible header, usually `text.common.actions`.
- User-facing text must go through locale data.
- Problem statements and the home notice are admin-managed trusted Markdown and may render raw HTML; user-generated Markdown must not.
- Use “用户” and “用户组” consistently in the admin UI; do not mix in “成员”.

## Product UI

- Home page order: notice, heatmap, latest problems/assignments/contests.
- The notice is edited inline on the home page, not in admin settings.
- Management actions should live near the business object: problems on problem pages, assignments on assignment pages, contests on contest pages, discussions on discussion pages.
- Editable entity list pages use modal create/edit forms; entity detail pages edit in place inside the existing primary Card. Reuse the business form fields, not a generic schema-driven editor.
- Card titles use the native Card title typography rather than nested heading components. Assignment and contest detail Card headers keep long titles on one ellipsized line and group actions on the right.
- Assignment and contest lists use a compact status Tag. Contest lists and detail headers order status, colored contest kind, then title. Keep the relevant time or range in the status tooltip.
- Problem references should display as clickable `P{id} {title}` under a column named “题目”.
- User references are clickable usernames with avatars.
- Markdown uploads must use server-returned relative URLs. Rendering may rewrite relative asset URLs to API URLs.
- Code highlighting matches the configured language `id` through CodeMirror language data; unknown languages fall back to plain text.

## Testing

- Vitest is for stable pure logic and small key components. It does not replace browser walkthrough.
- For pages touching permissions, visibility, assignments, contests, submissions, and statistics, rely on backend regression tests plus browser smoke checks.
