package app

import "context"

type clientIPKey struct{}

func withClientIP(ctx context.Context, ip string) context.Context {
	return context.WithValue(ctx, clientIPKey{}, ip)
}
func auditIP(ctx context.Context) string { ip, _ := ctx.Value(clientIPKey{}).(string); return ip }
