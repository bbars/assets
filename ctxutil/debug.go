package ctxutil

import (
	"context"
	"os"
)

func SetDebugAuto(parent context.Context) context.Context {
	debug := false
	if v, ok := os.LookupEnv("DEBUG"); ok {
		debug = v != "" && v != "0"
	}
	return SetDebug(parent, debug)
}

func SetDebug(parent context.Context, debug bool) context.Context {
	return context.WithValue(
		parent,
		"debug",
		debug,
	)
}

func IsDebug(ctx context.Context) bool {
	if v := ctx.Value("debug"); v == nil {
		return false
	} else {
		if vb, ok := v.(bool); !ok {
			return false
		} else {
			return vb
		}
	}
}
