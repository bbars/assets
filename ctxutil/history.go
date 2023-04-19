package ctxutil

import "context"

const HistoryKey = "prevContext"

func Push(ctx context.Context) context.Context {
	return context.WithValue(ctx, HistoryKey, ctx)
}

func Pop(ctx context.Context) context.Context {
	prevContext := ctx.Value(HistoryKey)
	if prevContext == nil {
		return ctx
	}
	return prevContext.(context.Context)
}
