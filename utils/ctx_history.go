package utils

import "context"

const ContextHistoryKey = "prevContext"

func ContextPush(ctx context.Context) context.Context {
	return context.WithValue(ctx, ContextHistoryKey, ctx)
}

func ContextPop(ctx context.Context) context.Context {
	prevContext := ctx.Value(ContextHistoryKey)
	if prevContext == nil {
		return ctx
	}
	return prevContext.(context.Context)
}
