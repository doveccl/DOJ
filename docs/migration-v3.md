# v3 JSON Migration

This importer expects MongoDB exports from the v3 branch and writes into the new PostgreSQL schema.

```bash
bun run db:migrate
bun run db:seed

mongoexport --db doj --collection users --jsonArray --out /tmp/doj-v3/users.json
mongoexport --db doj --collection problems --jsonArray --out /tmp/doj-v3/problems.json
mongoexport --db doj --collection contests --jsonArray --out /tmp/doj-v3/contests.json
mongoexport --db doj --collection submissions --jsonArray --out /tmp/doj-v3/submissions.json
mongoexport --db doj --collection posts --jsonArray --out /tmp/doj-v3/posts.json

MIGRATE_V3_EXPORT_DIR=/tmp/doj-v3 \
MIGRATE_V3_DEFAULT_PASSWORD='change-after-login' \
MIGRATE_V3_LANGUAGE_MAP='{"2":"py"}' \
bun run migrate:v3-json
```

Notes:

- Old Mongo `_id` values are stored in `legacyId` where the new table has that column.
- If `MIGRATE_V3_DEFAULT_PASSWORD` is set, all imported users receive a fresh Argon2id password. If omitted, the script keeps the v3 password hash as-is.
- v3 language IDs must be mapped to configured language IDs. The default only imports Python (`2 -> py`).
- v3 problem data archives are not imported yet; the script migrates statements, limits, tags, counters, contests, submissions, submission cases, and posts.
- Run against a database snapshot first. The importer skips rows already found by `legacyId`, but it is still meant as an operator-reviewed migration tool.
