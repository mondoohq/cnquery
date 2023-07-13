package common

import "context"

type ContextInitializer interface {
	InitCtx(ctx context.Context) context.Context
}
