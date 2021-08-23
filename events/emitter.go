package events

import (
	"fmt"
	"reflect"
	"sync"
)

type EventEmitter interface {
	Emit(topic string, args ...interface{})
	On(topic string, handler interface{})
	Once(topic string, handler interface{})
}

type Emitter struct {
	// Map topics to callbacks
	mu        sync.Mutex
	listeners map[string]*Handlers
}

// Contains some type info so that we know whether subsequent
// handlers are valid
// Also contains the handlers themselves
type Handlers struct {
	in    []reflect.Type  // Lists the types of the parameters to the handler function
	funcs []reflect.Value // The handler functions
}

func NewEmitter() *Emitter {
	return &Emitter{
		listeners: make(map[string]*Handlers),
	}
}

func (e *Emitter) On(topic string, handler interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()

	cbType := reflect.TypeOf(handler)
	if cbType.Kind() != reflect.Func {
		panicF("Expected handler to have kind Func, got: %s", cbType.Kind())
	}

	fmt.Printf("T[%s] handler func has NumIn: %d\n", topic, cbType.NumIn())

	// No handlers yet assigned to this topic
	if e.listeners[topic] == nil {
		inTypes := make([]reflect.Type, cbType.NumIn())

		// Add the input types for the handler function
		for i := 0; i < cbType.NumIn(); i++ {
			inTypes[i] = cbType.In(i)
		}

		e.listeners[topic] = &Handlers{
			in:    inTypes,
			funcs: make([]reflect.Value, 0),
		}
	} else {
		// We have previous handlers assigned to this topic.
		// Make sure the new handler matches the inTypes:
		numIn := cbType.NumIn()
		if numIn != len(e.listeners[topic].in) {
			panicF("T[%s] new handler wrong argument count. Expected %d; got %d", topic, len(e.listeners[topic].in), numIn)
		}

		for i := 0; i < cbType.NumIn(); i++ {
			if cbType.In(i).Kind() != e.listeners[topic].in[i].Kind() {
				panicF("T[%s] new handler invalid argument at position %d. Expected %s; got %s", topic, i, e.listeners[topic].in[i].Kind(), cbType.In(i).Kind())
			}
		}
	}

	// Add the handler to the topic:
	e.listeners[topic].funcs = append(e.listeners[topic].funcs, reflect.ValueOf(handler))
}

func (e *Emitter) Emit(topic string, args ...interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()

	// Emitting to no listeners! Return an error
	if e.listeners[topic] == nil {
		panicF("T[%s] has no listeners to emit to", topic)
	}

	fmt.Printf("T[%s] emitting with %d args\n", topic, len(args))

	// Construct input args and make sure the types in args match
	// the types specified by the handlers:
	inArgs := make([]reflect.Value, len(args))

	if len(args) != len(e.listeners[topic].in) {
		panicF("T[%s] has %d input args; got %d", topic, len(e.listeners[topic].in), len(args))
	}

	for i, arg := range args {
		t := reflect.TypeOf(arg)
		if t.Kind() != e.listeners[topic].in[i].Kind() {
			panicF("T[%s] invalid argument at position %d. Expected %s; got %s", topic, i, e.listeners[topic].in[i].Kind(), t.Kind())
		}

		inArgs[i] = reflect.ValueOf(arg)
	}

	// Call each handler:
	for _, fn := range e.listeners[topic].funcs {
		go fn.Call(inArgs) // for now, return values are ignored
	}
}

func panicF(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
