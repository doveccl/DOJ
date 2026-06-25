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
- Do not specify component `size` by default. In dense nested admin surfaces such as a modal containing cards/tables/tool buttons, use `small` consistently inside that surface.
- Use Tag for row state when table row styling would require custom CSS.
- Table action columns must have a visible header, usually `text.common.actions`.
- User-facing text must go through locale data.
- Use “用户” and “用户组” consistently in the admin UI; do not mix in “成员”.

## Product UI

- Home page order: notice, heatmap, latest problems/assignments/contests.
- The notice is edited inline on the home page, not in admin settings.
- Management actions should live near the business object: problems on problem pages, assignments on assignment pages, contests on contest pages, discussions on discussion pages.
- Problem references should display as clickable `P{id} {title}` under a column named “题目”.
- User references are clickable usernames; only the rank page shows avatars in the user column.
- Markdown uploads must use server-returned relative URLs. Rendering may rewrite relative asset URLs to API URLs.
- Code highlighting matches the configured language `id` through CodeMirror language data; unknown languages fall back to plain text.

## Testing

- Vitest is for stable pure logic and small key components. It does not replace browser walkthrough.
- For pages touching permissions, visibility, assignments, contests, submissions, and statistics, rely on backend regression tests plus browser smoke checks.
