# cronwhy

[![CI](https://github.com/your-username/cronwhy/actions/workflows/ci.yml/badge.svg)](https://github.com/your-username/cronwhy/actions/workflows/ci.yml)
[![Go Reference](https://pkg.go.dev/badge/github.com/your-username/cronwhy.svg)](https://pkg.go.dev/github.com/your-username/cronwhy)
[![License: MIT](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)

**Explain cron expressions and lint whole crontab files.**

Tools like crontab.guru and cronstrue explain one expression at a time. Real
crontabs break in other ways: a `%` that silently truncates a command, a
"first Monday of the month" job that runs on about ten days a month, seven jobs that
all start at midnight, a 02:30 backup that disappears when the clocks
change. `cronwhy` reads the whole file and finds these problems.

- Plain-English explanations and the next run times, with timezone and DST support
- 11 lint rules with line numbers and suggested fixes
- `golangci-lint`-style text output or a stable JSON schema for CI and editors
- Follows Vixie cron semantics, including the day-of-month / day-of-week OR rule
- A single static binary that uses only the Go standard library

## Install

```sh
go install github.com/your-username/cronwhy/cmd/cronwhy@latest
```

Or from source:

```sh
git clone https://github.com/your-username/cronwhy && cd cronwhy && make build
```

## Usage

### `explain`

```console
$ cronwhy explain "*/15 9-17 * * 1-5" -n 5 --from "2026-10-27 20:00" --tz Europe/Istanbul
*/15 9-17 * * 1-5
At every 15th minute, from 09:00 through 17:59, Monday through Friday

  minute        */15       0,15,30,45
  hour          9-17       9-17
  day-of-month  *          1-31
  month         *          1-12
  day-of-week   1-5        1-5

Next 5 runs (Europe/Istanbul, from Tue 2026-10-27 20:00):
  Wed 2026-10-28 09:00 +03  in 13h 0m
  Wed 2026-10-28 09:15 +03  in 13h 15m
  Wed 2026-10-28 09:30 +03  in 13h 30m
  Wed 2026-10-28 09:45 +03  in 13h 45m
  Wed 2026-10-28 10:00 +03  in 14h 0m
```

The day-of-month / day-of-week trap:

```console
$ cronwhy explain "0 8 1-7 * MON" -n 4 --from "2026-10-27 20:00" --tz UTC
0 8 1-7 * MON
At 08:00, on every day-of-month from 1 through 7 or Monday

  minute        0          0
  hour          8          8
  day-of-month  1-7        1-7
  month         *          1-12
  day-of-week   MON        1

Note: day-of-month and day-of-week are both restricted, so cron runs the job
when EITHER one matches (Vixie OR rule), not only when both match.

Next 4 runs (UTC, from Tue 2026-10-27 20:00):
  Sun 2026-11-01 08:00 UTC  in 4d 12h
  Mon 2026-11-02 08:00 UTC  in 5d 12h
  Tue 2026-11-03 08:00 UTC  in 6d 12h
  Wed 2026-11-04 08:00 UTC  in 7d 12h
```

Invalid expressions report the field and the reason:

```console
$ cronwhy explain "5/15 * * * *"
cronwhy: invalid expression "5/15 * * * *"
  minute field "5/15": a step on a single value (5/15) is not supported by Vixie cron
  hint: use a range: 5-59/15
```

### `next`

Prints only the times (RFC 3339), one per line, so the output is easy to use in scripts:

```console
$ cronwhy next "@weekly" -n 3 --from "2026-10-27 20:00" --tz America/New_York
2026-11-01T00:00:00-04:00
2026-11-08T00:00:00-05:00
2026-11-15T00:00:00-05:00
```

### `lint`

```
cronwhy lint [file... | -] [--system] [--format text|json] [--strict]
             [--tz ZONE] [--window 7d] [--pile-up 5] [--from "YYYY-MM-DD HH:MM"]
```

With no file, `lint` reads `crontab -l`. With `-`, it reads stdin. `--system` expects the
`/etc/crontab` and `/etc/cron.d` format, where the sixth field is the user.

```console
$ cronwhy lint testdata/broken.crontab --from "2026-10-27 20:00"
testdata/broken.crontab:6: warning: 7 jobs start in the same minute at 00:00 (lines 6, 7, 8, 9, 10, 20, 21); this happens 7 time(s) in the 7d window; the top of the hour (especially midnight) is the busiest minute on most systems (pile-up)
    0 0 * * * /opt/app/bin/rotate-logs >> /var/log/rotate.log 2>&1
    fix: stagger the start minutes (e.g. 3, 17, 34, 48) or add a random delay such as: sleep $((RANDOM \% 300)); cmd
testdata/broken.crontab:10: warning: in America/New_York when DST starts clocks jump from 02:00 to 03:00, so runs at 02:00 may be skipped; when DST ends clocks go back from 02:00 to 01:00, so runs at 01:00 may happen twice (behaviour depends on the cron implementation) (dst-risk)
    0 * * * * /opt/app/bin/sync-metrics >> /var/log/metrics.log 2>&1
    fix: schedule the job outside the transition window (e.g. 03:30 or later) or set CRON_TZ=UTC for it
testdata/broken.crontab:13: error: 3 unescaped % in the command; cron turns the first one into a newline and sends the rest to stdin, so the shell only runs: /usr/bin/pg_dump app > /backups/app-$(date + (unescaped-percent)
    30 2 * * * /usr/bin/pg_dump app > /backups/app-$(date +%Y-%m-%d).sql
    fix: escape it: /usr/bin/pg_dump app > /backups/app-$(date +\%Y-\%m-\%d).sql
testdata/broken.crontab:13: warning: in America/New_York when DST starts clocks jump from 02:00 to 03:00, so runs at 02:30 may be skipped (behaviour depends on the cron implementation) (dst-risk)
    30 2 * * * /usr/bin/pg_dump app > /backups/app-$(date +%Y-%m-%d).sql
    fix: schedule the job outside the transition window (e.g. 03:30 or later) or set CRON_TZ=UTC for it
testdata/broken.crontab:14: error: schedule "0 0 31 2 *" can never fire: no selected month has the selected day-of-month (impossible-date)
    0 0 31 2 * /opt/app/bin/leap-report >> /var/log/leap.log 2>&1
    fix: pick a day that exists in the selected months (February has at most 29 days; April, June, September and November have 30)
testdata/broken.crontab:17: warning: day-of-month "1-7" and day-of-week "MON" are both restricted, so cron runs the job when EITHER matches (on day-of-month 1-7 and also on every day-of-week MON), not only when both match (dom-dow-or)
    0 8 1-7 * MON /opt/app/bin/monthly-report >> /var/log/report.log 2>&1
    fix: to require both, set day-of-week to * and test the weekday in the command, e.g. [ "$(date +\%u)" = 1 ] && cmd
testdata/broken.crontab:20: warning: in America/New_York when DST starts clocks jump from 02:00 to 03:00, so runs at 02:00 ... 02:56 (9 runs) may be skipped; when DST ends clocks go back from 02:00 to 01:00, so runs at 01:00 ... 01:56 (9 runs) may happen twice (behaviour depends on the cron implementation) (dst-risk)
    */7 * * * * /opt/app/bin/poll-queue >> /var/log/poll.log 2>&1
    fix: schedule the job outside the transition window (e.g. 03:30 or later) or set CRON_TZ=UTC for it
testdata/broken.crontab:20: warning: */7 in the minute field fires at 0,7,14,21,28,35,42,49,56; after 56 it wraps to 0, a gap of 4 minute(s) instead of 7 (uneven-step)
    */7 * * * * /opt/app/bin/poll-queue >> /var/log/poll.log 2>&1
    fix: use a step that divides 60 evenly (*/2, */3, */4, */5, */6, */10, */12, */15, */20, */30) or list the exact values
testdata/broken.crontab:21: warning: in America/New_York when DST starts clocks jump from 02:00 to 03:00, so runs at 02:00 ... 02:59 (60 runs) may be skipped; when DST ends clocks go back from 02:00 to 01:00, so runs at 01:00 ... 01:59 (60 runs) may happen twice (behaviour depends on the cron implementation) (dst-risk)
    * * * * * /opt/app/bin/heartbeat >> /var/log/heartbeat.log 2>&1
    fix: schedule the job outside the transition window (e.g. 03:30 or later) or set CRON_TZ=UTC for it
testdata/broken.crontab:21: info: job runs every minute (up to 1440 times a day) (every-minute)
    * * * * * /opt/app/bin/heartbeat >> /var/log/heartbeat.log 2>&1
    fix: if this is intended, make sure the job finishes well within a minute (or use flock); otherwise pick a specific minute or a step like */5
testdata/broken.crontab:24: error: hour field "25": value 25 out of range 0-23 (invalid-field)
    0 25 * * * /opt/app/bin/legacy-cleanup >> /var/log/legacy.log 2>&1
    fix: valid ranges: minute 0-59, hour 0-23, day-of-month 1-31, month 1-12 or JAN-DEC, day-of-week 0-7 or SUN-SAT
testdata/broken.crontab:25: warning: "python3" is not an absolute path and PATH is not set; cron's default PATH is usually only /usr/bin:/bin (relative-path)
    15 4 * * 1-5 python3 /opt/app/scripts/export.py >> /var/log/export.log 2>&1
    fix: use the full path (find it with: command -v python3) or add PATH=/usr/local/bin:/usr/bin:/bin at the top of the crontab
testdata/broken.crontab:26: info: output is not redirected and MAILTO is not set; cron mails it to the local user, where it piles up unread or is lost if no MTA is installed (no-output-handling)
    45 5 * * * /opt/app/bin/send-digest
    fix: append ">> /var/log/job.log 2>&1", or set MAILTO=you@example.com (MAILTO="" to discard output)
testdata/broken.crontab:27: warning: same schedule and command as line 26; the job runs twice every time (duplicate-job)
    45 5 * * * /opt/app/bin/send-digest
    fix: remove one of the two lines
testdata/broken.crontab:27: info: output is not redirected and MAILTO is not set; cron mails it to the local user, where it piles up unread or is lost if no MTA is installed (no-output-handling)
    45 5 * * * /opt/app/bin/send-digest
    fix: append ">> /var/log/job.log 2>&1", or set MAILTO=you@example.com (MAILTO="" to discard output)
testdata/broken.crontab:28: error: day-of-month field "L": "L" is a Quartz/AWS extension (L/W); this is not supported by standard cron (invalid-field)
    0 0 L * * /opt/app/bin/month-end >> /var/log/month-end.log 2>&1
    fix: for the last day of the month use 28-31 and test it in the command: [ "$(date -d tomorrow +\%d)" = 01 ] && cmd

testdata/broken.crontab: 4 errors, 9 warnings, 3 info (15 jobs checked)
```

```console
$ cronwhy lint testdata/clean.crontab --strict
ok testdata/clean.crontab: no problems found (7 jobs checked)
```

Output is colored only when stdout is a terminal and `NO_COLOR` is not set.

#### JSON output

```console
$ cronwhy lint testdata/readme.crontab --format json
{
  "version": 1,
  "summary": {
    "files": 1,
    "jobs": 2,
    "errors": 1,
    "warnings": 1,
    "info": 0
  },
  "findings": [
    {
      "rule": "unescaped-percent",
      "severity": "error",
      "file": "testdata/readme.crontab",
      "line": 2,
      "message": "1 unescaped % in the command; cron turns the first one into a newline and sends the rest to stdin, so the shell only runs: /usr/bin/tar czf /backups/home-$(date +",
      "fix": "escape it: /usr/bin/tar czf /backups/home-$(date +\\%F).tgz /home",
      "source": "0 3 * * * /usr/bin/tar czf /backups/home-$(date +%F).tgz /home"
    },
    {
      "rule": "uneven-step",
      "severity": "warning",
      "file": "testdata/readme.crontab",
      "line": 3,
      "message": "*/7 in the minute field fires at 0,7,14,21,28,35,42,49,56; after 56 it wraps to 0, a gap of 4 minute(s) instead of 7",
      "fix": "use a step that divides 60 evenly (*/2, */3, */4, */5, */6, */10, */12, */15, */20, */30) or list the exact values",
      "source": "*/7 * * * * /opt/app/bin/poll >> /var/log/poll.log 2>&1"
    }
  ]
}
```

## Rules

| ID | Severity | What it catches |
|----|----------|-----------------|
| `invalid-field` | error | Out-of-range values, reversed ranges, bad steps, unknown names, Quartz-only syntax (`L`, `W`, `#`, `?`), malformed lines |
| `impossible-date` | error | Schedules that can never fire, e.g. `0 0 31 2 *` |
| `unescaped-percent` | error | `%` in a command that is not written as `\%`; cron turns it into a newline |
| `dom-dow-or` | warning | Day-of-month and day-of-week both restricted: cron fires when **either** matches |
| `uneven-step` | warning | Steps such as `*/7` that leave a short gap when they wrap around (56 → 0) |
| `dst-risk` | warning | Runs that fall inside a DST transition in the job's timezone (skipped or doubled) |
| `pile-up` | warning | N or more jobs starting in the same minute, found by simulating `--window` |
| `duplicate-job` | warning | Same schedule (`@daily` = `0 0 * * *`) and same command twice |
| `relative-path` | warning | Command is not an absolute path and `PATH` is not set |
| `every-minute` | info | Jobs that run every minute |
| `no-output-handling` | info | No output redirection and no `MAILTO` |

Timezone used by time-based rules: `CRON_TZ` above the job if set, otherwise `--tz`,
otherwise the system's local zone.

## Exit codes

| Code | Meaning |
|------|---------|
| 0 | No findings, or only warnings/info without `--strict` |
| 1 | Only warnings/info found and `--strict` was given |
| 2 | At least one error (also: invalid expression for `explain`/`next`) |
| 3 | Usage or read error |

## JSON schema (version 1)

```jsonc
{
  "version": 1,                 // bumped only on breaking changes
  "summary": {
    "files": 1, "jobs": 2,      // files linted, job lines seen
    "errors": 1, "warnings": 1, "info": 0
  },
  "findings": [                 // always an array, sorted by file, line, severity
    {
      "rule": "unescaped-percent",       // rule ID from the table above
      "severity": "error",               // "error" | "warning" | "info"
      "file": "testdata/readme.crontab", // "<stdin>" or "crontab -l" for those inputs
      "line": 2,                         // 1-based
      "message": "…",
      "fix": "…",                        // suggested fix, may be ""
      "source": "…"                      // the trimmed source line
    }
  ]
}
```

## Integrations

### pre-commit

```yaml
# .pre-commit-config.yaml
repos:
  - repo: https://github.com/your-username/cronwhy
    rev: v0.1.0
    hooks:
      - id: cronwhy
```

The hook runs on files named `crontab`, `*.cron` and `*.crontab`. Pass
`args: [--system]` for `/etc/cron.d` style files.

### GitHub Actions

```yaml
# .github/workflows/crontab-lint.yml
name: crontab lint
on: [push, pull_request]
jobs:
  cronwhy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-go@v5
        with:
          go-version: "1.22"
      - run: go install github.com/your-username/cronwhy/cmd/cronwhy@latest
      - run: cronwhy lint --strict deploy/crontab
```

## Behavior notes

- Follows **Vixie cron**: `0` and `7` both mean Sunday. A step on a single value (`5/15`) is
  rejected. If day-of-month **or** day-of-week starts with `*`, both fields must match;
  otherwise either one is enough.
- `Next` jumps by month, day, hour and minute instead of checking every minute. A
  property test compares it with a minute-by-minute reference implementation across
  DST transitions in `Europe/Istanbul`, `America/New_York` and `Europe/London`.
- Times are matched on the local wall clock. A time skipped by DST never fires, and
  a time that happens twice fires twice. Real cron daemons handle these cases in
  different ways, which is why `dst-risk` exists.

## Development

```sh
make test   # go test ./... -race
make lint   # gofmt + go vet
make fuzz   # fuzz the parser for 30s
go test ./cmd/cronwhy -update   # regenerate golden files and README examples
```

See [CLAUDE.md](CLAUDE.md) for the architecture and how to add a rule.

## License

[MIT](LICENSE)
