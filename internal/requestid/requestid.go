// Package requestid holds the request-ID context key and header name as a
// standalone package with no other dependencies. Both middleware (which
// sets the ID) and response (which reads it for error logging) depend on
// this instead of on each other, which would otherwise be an import cycle.
package requestid

import "context"

type contextKey string

const key contextKey = "request_id"

const Header = "X-Request-ID"

func Set(ctx context.Context, id string) context.Context {
	return context.WithValue(ctx, key, id)
}

func FromContext(ctx context.Context) string {
	id, _ := ctx.Value(key).(string)
	return id
}
