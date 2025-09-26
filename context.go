package main

import (
	"context"
)

type ctxKey struct{}

var wallsKey = ctxKey{}

func getWalls(ctx context.Context) *Walls {
	walls, ok := ctx.Value(wallsKey).(*Walls)
	if !ok {
		panic("no walls instance in context")
	}
	return walls
}

func setWalls(ctx context.Context, walls *Walls) context.Context {
	return context.WithValue(ctx, wallsKey, walls)
}
