package utils

import (
	"context"
	"os"
)

const ContextDebugKey = "debug"

func ContextSetDebugAuto(parent context.Context) context.Context {
	debug := false
	if v, ok := os.LookupEnv("DEBUG"); ok {
		debug = v != "" && v != "0"
	}
	return ContextSetDebug(parent, debug)
}

func ContextSetDebug(parent context.Context, debug bool) context.Context {
	return context.WithValue(
		parent,
		ContextDebugKey,
		debug,
	)
}

func ContextIsDebug(ctx context.Context) bool {
	if v := ctx.Value(ContextDebugKey); v == nil {
		return false
	} else {
		if vb, ok := v.(bool); !ok {
			return false
		} else {
			return vb
		}
	}
}
