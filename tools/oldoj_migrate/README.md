# oldoj_migrate

Low-impact helper for migrating real problem data from the legacy Mongo-based
OJ into the test deployment.

The tool is intentionally batch-oriented:

- `plan` reads only lightweight metadata from the old Mongo and test Postgres.
- `migrate` streams one legacy GridFS zip at a time over SSH to a local temp
  file, uploads extracted `data/*.in` and `data/*.out` files from this machine
  to object storage, then upserts one Postgres problem row.
- `cleanup` soft-deletes test problems that have no data cases. It is dry-run
  by default.
- `verify` checks the active problem count with data cases and reports any
  active problems without cases.

Example:

```sh
export STORAGE='https://...'
go run ./tools/oldoj_migrate plan -limit 100 -timeout 20s
go run ./tools/oldoj_migrate migrate -limit 5 -sleep 5s -timeout 30s --apply
go run ./tools/oldoj_migrate cleanup -timeout 20s --apply
go run ./tools/oldoj_migrate verify -min-data 100 -max-empty 0 -timeout 20s
```

Do not run a large `-limit` while the test site is being manually checked.
Do not add remote tar/docker-cp/export batching here; the VPS is too small for
that style of migration.
