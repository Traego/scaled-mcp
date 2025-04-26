package auth

import (
	"context"
	"github.com/traego/scaled-mcp/pkg/utils"
)

type AuthInfo interface {
	GetPrincipalId() string
}

func GetAuthInfo(ctx context.Context) AuthInfo {
	a := ctx.Value(utils.AuthInfoCtx)
	if a == nil {
		return nil
	} else {
		return a.(AuthInfo)
	}
}

func SetAuthInfo(ctx context.Context, ai AuthInfo) context.Context {
	return context.WithValue(ctx, utils.AuthInfoCtx, ai)
}
