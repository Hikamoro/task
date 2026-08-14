package middleware

import "context"

type ctxKey int

const userIDKey ctxKey = iota

func WithUserID(ctx context.Context, id int64) context.Context {
	return context.WithValue(ctx, userIDKey, id)
}

func UserID(ctx context.Context) (int64, bool) {
	id, ok := ctx.Value(userIDKey).(int64)
	return id, ok
}