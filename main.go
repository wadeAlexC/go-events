package main

import (
	"context"
	"fmt"
	"time"

	"github.com/wadeAlexC/go-events/events"
)

func main() {
	thing := events.NewEmitter()

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Second)

	a := uint64(1)

	for i := 0; i < 5; i++ {
		thing.On("ready", func(val uint64) {
			fmt.Printf("CB: %d\n", val)
			fmt.Printf("Val of a: %d\n", a)
			a++
		})
	}

	thing.On("ready", func(val uint64) {
		if a > 5 {
			cancel()
		}
	})

	err := thing.Emit("ready", uint64(5))

	if err != nil {
		fmt.Printf("Error at Emit: %v\n", err)
	}

	err = thing.Emit("ready", uint64(6))

	if err != nil {
		fmt.Printf("Error at Emit: %v\n", err)
	}

	for {
		select {
		case <-ctx.Done():
			cancel()
			return
		}
	}
}
