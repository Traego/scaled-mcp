package actors

import (
	"context"
	"github.com/tochemey/goakt/v3/actor"
)

type RootActor struct {
}

func (r RootActor) PreStart(ctx context.Context) error {
	return nil
}

func (r RootActor) Receive(ctx *actor.ReceiveContext) {
}

func (r RootActor) PostStop(ctx context.Context) error {
	return nil
}

func NewRootActor() actor.Actor {
	return &RootActor{}
}

var _ actor.Actor = (*RootActor)(nil)
