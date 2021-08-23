package main

import (
	"context"
	"fmt"
	"time"

	"github.com/wadeAlexC/go-events/events"
)

type Thing struct {
	*events.Emitter
	name string
	val  uint64
}

func NewThing() *Thing {
	return &Thing{
		Emitter: events.NewEmitter(),
	}
}

func (t *Thing) start(name string, val uint64) {
	t.name = name
	t.val = val
	t.Emit("named", t.name)
	t.Emit("valued", t.val)
	t.Emit("close")
}

func (t *Thing) prepare() {

	ctx, _ := context.WithTimeout(context.Background(), 5*time.Second)

	for {
		select {
		case <-ctx.Done():
			t.Emit("ready", "alex", uint64(5))
			return
		}
	}
}

func main() {
	thing := NewThing()

	thing.On("ready", thing.start)

	thing.On("named", func(name string) {
		fmt.Printf("Thing named %s\n", name)
	})

	thing.On("valued", func(val uint64) {
		fmt.Printf("Thing named %d\n", val)
	})

	thing.On("close", func() {
		fmt.Printf("Thing stopped!\n")
	})

	thing.prepare()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)

	for {
		select {
		case <-ctx.Done():
			cancel()
			return
		}
	}
}
