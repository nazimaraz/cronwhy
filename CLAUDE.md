# cronwhy

CLI that explains cron expressions and lints whole crontab files. Go 1.22+, standard
library only, Vixie cron semantics. Out of scope: running jobs, a daemon, network access, LLMs.

## Architecture

```
cmd/cronwhy/        main.go (dispatch, interspersed flag parsing), explain.go, lint.go.
                    run(args, *env) int is the testable entry point; env injects
                    stdin/stdout, clock, NO_COLOR, TTY and `crontab -l`.
internal/cronexpr/  Parse -> *Schedule with 5 bitset Fields (each keeps its Terms for
                    lint). Field.Star = text starts with "*" (Vixie DOM/DOW AND/OR rule).
                    Next/Prev jump month/day/hour/minute on wall-clock time, bounded by
                    searchYears; Possible() is precomputed at parse time.
internal/explain/   Describe(schedule) -> English; Relative, Ordinal, CompactValues.
internal/crontab/   Parse(reader, name, Options{System}) -> *File. Env lines apply to the
                    entries that follow them (Entry.Env snapshot). Bad lines become
                    Entries with Err set and do not stop parsing.
internal/lint/      Rule interface, registry, Run(). One rule per rule_<id>.go file.
internal/report/    Text (golangci-lint style, ANSI only on TTY and without NO_COLOR) and JSON
                    (schema documented in README; bump SchemaVersion on breaking changes).
testdata/           broken/clean/system crontabs, golden outputs, readme.crontab.
```

## Commands

```sh
make build                      # ./cronwhy
make test                       # go test ./... -race
make lint                       # gofmt check + go vet
make fuzz                       # 30s parser fuzzing
go test ./cmd/cronwhy -update   # regenerate testdata/broken.golden.* and README examples
```

README console blocks (```` ```console ```` + `$ cronwhy ...`) are run by
`cmd/cronwhy/readme_test.go`. Do not edit their output by hand. Run `-update` instead.

## Adding a lint rule

1. Create `internal/lint/rule_<id>.go`:
   ```go
   func init() { register(myRule{meta{"my-rule", Warning, "One-line summary"}}) }
   type myRule struct{ meta }
   func (r myRule) Check(ctx *Context) []Finding { /* r.finding(line, fix, fmt, args...) */ }
   ```
   Use `ctx.Valid()` for parsed entries and `ctx.Location(e)` for the effective timezone.
   Keep messages deterministic (no absolute dates) because golden files depend on them.
2. Add `rule_<id>_test.go` with `runRuleCases(t, "my-rule", []ruleCase{...})`, covering
   both cases that trigger the rule and cases that do not.
3. Add the rule ID to `TestRegistry` in `lint_test.go`.
4. Add a line to `testdata/broken.crontab` that triggers it (`TestBrokenTriggersEveryRule`
   enforces this), and make sure `testdata/clean.crontab --strict` still exits 0.
5. Add a row to the rules table in README.md and run `go test ./cmd/cronwhy -update`.
