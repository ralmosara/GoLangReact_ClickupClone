# Backup & disaster recovery

This document is the runbook for backing up Postgres and restoring the
system from cold. The aim is the targets below; code in this repository
does not yet automate them — operators run the commands.

## Targets

| Metric | Target | Notes |
| ------ | ------ | ----- |
| RPO (recovery point objective) | ≤ 5 min | WAL archive interval cap |
| RTO (recovery time objective) | ≤ 30 min | from "incident declared" to "writes accepted" |
| Backup retention | 30 days base + 7 days WAL | shipped to S3-compatible storage |
| Restore drill cadence | quarterly | logged in the on-call calendar |

## Components

1. **Base backups + WAL archiving** via [pgBackRest](https://pgbackrest.org).
2. **Object-store target** — any S3-compatible bucket (AWS S3, R2, MinIO).
3. **Attachment store** — the Docker volume `backend-storage` (or the
   `STORAGE_DIR` directory) snapshot to the same bucket nightly.

## Reference pgBackRest config

Place at `/etc/pgbackrest/pgbackrest.conf` on the Postgres host:

```ini
[global]
repo1-type=s3
repo1-s3-bucket=clickup-pg-backups
repo1-s3-endpoint=s3.amazonaws.com
repo1-s3-region=us-east-1
repo1-s3-key=AKIA...
repo1-s3-key-secret=...
repo1-retention-full=4        # keep 4 weeks of weekly full backups
repo1-retention-diff=14       # keep 14 daily diff backups
repo1-cipher-type=aes-256-cbc
repo1-cipher-pass=...         # generated with: openssl rand -base64 48
process-max=4
log-level-console=info
log-level-file=detail
start-fast=y

[clickup]
pg1-path=/var/lib/postgresql/16/main
pg1-port=5432
pg1-host-user=postgres
```

In `postgresql.conf`:

```ini
archive_mode = on
archive_command = 'pgbackrest --stanza=clickup archive-push %p'
archive_timeout = 60          # cap RPO at one minute even on idle
max_wal_senders = 3
wal_level = replica
```

## Daily operations

| Task | Command | Schedule |
| ---- | ------- | -------- |
| Full backup | `pgbackrest --stanza=clickup --type=full backup` | Sunday 02:00 UTC |
| Differential | `pgbackrest --stanza=clickup --type=diff backup` | Mon–Sat 02:00 UTC |
| Verify | `pgbackrest --stanza=clickup verify` | Daily after backup |
| Attachment snapshot | `tar -czf /backup/storage-$(date +%F).tgz $STORAGE_DIR && aws s3 cp ...` | Nightly 03:00 UTC |

A cron / systemd-timer fragment lives at `infra/cron/pgbackrest.cron`
(create when this is automated; deferred from the current PR).

## Restore drill

Quarterly drill, run on a non-production replica. The drill is **green**
when steps 1–6 complete in under 30 minutes and the integrity check passes.

1. Stand up an empty Postgres host with the same major version.
2. Install pgbackrest with the production config (read-only credentials).
3. `pgbackrest --stanza=clickup restore --type=time --target='2025-01-15 12:00:00'`
4. Start Postgres; confirm it reaches a consistent state.
5. Run `SELECT MAX(version) FROM schema_migrations` — must match the
   production schema version captured at drill start.
6. Run the smoke-test integration suite (`go test ./internal/integration_test/...`)
   pointed at the restored instance.
7. Document drift, drift duration, and any operator surprises in the
   on-call log.

## High-availability (out of scope this phase)

The roadmap calls for synchronous replication via Patroni + etcd or a
managed offering (AWS RDS Multi-AZ, Cloud SQL HA). That work is tracked
separately; this document covers backups only. Until HA lands, a
single-instance database is the published reliability ceiling and the
`/api/v1/*` SLO above is contingent on a < 30-min restore.

## Encryption posture

- **At rest**: backups are encrypted with `repo1-cipher-pass` before
  upload. The bucket additionally enforces SSE-KMS as a defence in depth.
- **In transit**: TLS to S3; pgBackRest verifies CA bundle.
- **Application secrets**: the credentials vault uses AES-256-GCM with
  a 32-byte key from `CREDENTIALS_ENCRYPTION_KEY`. Backups of `credentials`
  rows are useless without that key — rotate it on a schedule (yearly)
  and document the previous keys in your KMS so historical decrypts are
  recoverable.
