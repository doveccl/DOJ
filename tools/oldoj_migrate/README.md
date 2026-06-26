# oldoj_migrate

Low-impact helper for migrating real problem data from the legacy Mongo-based
OJ into the test deployment.

The tool is intentionally batch-oriented:

- `plan` reads only lightweight metadata from the old Mongo and test Postgres.
- `migrate` streams one legacy GridFS zip at a time over SSH, uploads extracted
  `data/*.in` and `data/*.out` files from this machine to object storage, then
  upserts one Postgres problem row.
- `cleanup` soft-deletes test problems that have no data cases. It is dry-run
  by default.

Example:

```sh
export STORAGE='https://...'
go run ./tools/oldoj_migrate plan -limit 100 -timeout 20s
go run ./tools/oldoj_migrate migrate -limit 5 -sleep 5s -timeout 30s --apply
go run ./tools/oldoj_migrate cleanup -timeout 20s --apply
```

Do not run a large `-limit` while the test site is being manually checked.
