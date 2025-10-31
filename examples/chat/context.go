package main

import "context"

func getUsernameFromContext(ctx context.Context) string {
	val := ctx.Value("username")
	if val == nil {
		return ""
	}

	return val.(string)
}

func getConnectionIdFromContext(ctx context.Context) string {
	val := ctx.Value("connectionId")
	if val == nil {
		return ""
	}

	return val.(string)
}
