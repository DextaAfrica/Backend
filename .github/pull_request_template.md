## Summary

<!-- What changed, and why. -->

## Checklist

- [ ] `go build ./...`, `go vet ./...`, and `go test ./...` pass locally
- [ ] New/changed behavior has test coverage
- [ ] `.env.example` updated if a new environment variable was introduced
- [ ] Schema changes: a new `NNNNNN_description.up.sql` / `.down.sql` pair was added (never edit `000001_schema.*.sql` after it has shipped)
- [ ] `docs/ARCHITECTURE.md` updated if this changes layering, the error boundary, or the content model
