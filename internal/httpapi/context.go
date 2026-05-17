package httpapi

import "context"

type traceIDKey struct{}

func traceIDFromContext(ctx context.Context) string {
	traceID, _ := ctx.Value(traceIDKey{}).(string)
	return traceID
}
