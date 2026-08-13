package rconv2

import "context"

type socketConnection interface {
	getSocket() *socket
}

func DumpPlain[T any](ctx context.Context, c socketConnection, r T) (*string, error) {
	return execCommand[T, string](ctx, c.getSocket(), r)
}
