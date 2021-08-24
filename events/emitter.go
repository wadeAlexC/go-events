package events

import (
	"fmt"
	"reflect"
	"sync"
)

type EventEmitter interface {
	On(topic string, handler interface{})
	Once(topic string, handler interface{})

	Emit(topic string, args ...interface{})
	EmitSync(topic string, args ...interface{})
}

type Emitter struct {
	// Map topics to callbacks
	mu        sync.Mutex
	listeners map[string]*Handlers
}

var _ EventEmitter = (*Emitter)(nil)

// Contains some type info so that we know whether subsequent
// handlers are valid
// Also contains the handlers themselves
type Handlers struct {
	mu    sync.Mutex
	in    []reflect.Type // Lists the types of the parameters to the handler functions
	funcs []HandlerFunc  // The handler functions
}

type HandlerFunc struct {
	once bool
	f    reflect.Value
}

func NewEmitter() *Emitter {
	return &Emitter{
		listeners: make(map[string]*Handlers),
	}
}

/// REGISTERING CALLBACKS:
//
// On/Once register a callback function for a topic.
//
// Requirements:
// 1. The param types and positions of each subsequent callback
//    must be the same as the other callbacks registered for
//    the same topic.

// On registers a callback function for some topic
func (e *Emitter) On(topic string, handler interface{}) {
	e.addHandler(false, topic, handler)
}

// Once registers a callback function for some topic. The
// callback is removed after it has been invoked.
func (e *Emitter) Once(topic string, handler interface{}) {
	e.addHandler(true, topic, handler)
}

/// EMITTING EVENTS:
//
// Emit/EmitAsync call all registered callbacks for a topic,
// passing in args as input parameters.
//
// Callbacks are fired in the order they were registered.
//
// Requirements:
// 1. The types and positions of args must exactly match the
//    types and positions of the callback's input parameters.

// Emit fires callbacks asynchronously, spawning a goroutine
// for each callback.
func (e *Emitter) Emit(topic string, args ...interface{}) {
	e.callHandlers(false, topic, args...)
}

// EmitSync fires callbacks synchronously, waiting for each
// callback to return before firing the next one.
func (e *Emitter) EmitSync(topic string, args ...interface{}) {
	e.callHandlers(true, topic, args...)
}

func (e *Emitter) addHandler(doOnce bool, topic string, handler interface{}) {
	e.mu.Lock()
	defer e.mu.Unlock()

	cbType := reflect.TypeOf(handler)
	if cbType.Kind() != reflect.Func {
		panicF("Expected handler to have kind Func, got: %s", cbType.Kind())
	}

	// fmt.Printf("T[%s] handler func has NumIn: %d\n", topic, cbType.NumIn())

	handlers, exists := e.listeners[topic]

	// No handlers yet assigned to this topic
	if !exists {
		inTypes := make([]reflect.Type, cbType.NumIn())

		// Add the input types for the handler function
		for i := 0; i < cbType.NumIn(); i++ {
			inTypes[i] = cbType.In(i)
		}

		handlers = &Handlers{
			in:    inTypes,
			funcs: make([]HandlerFunc, 0),
		}

		e.listeners[topic] = handlers
	} else {
		// We have previous handlers assigned to this topic.
		// Make sure the new handler matches the inTypes:
		numIn := cbType.NumIn()
		if numIn != len(handlers.in) {
			panicF("T[%s] new handler wrong argument count. Expected %d; got %d", topic, len(handlers.in), numIn)
		}

		for i := 0; i < cbType.NumIn(); i++ {
			if cbType.In(i).Kind() != handlers.in[i].Kind() {
				panicF("T[%s] new handler invalid argument at position %d. Expected %s; got %s", topic, i, handlers.in[i].Kind(), cbType.In(i).Kind())
			}
		}
	}

	// Add the handler to the topic:
	handlers.funcs = append(handlers.funcs, HandlerFunc{
		once: doOnce,
		f:    reflect.ValueOf(handler),
	})
}

func (e *Emitter) callHandlers(doSync bool, topic string, args ...interface{}) {
	e.mu.Lock()
	// Emitting to no listeners! Do nothing.
	if _, exists := e.listeners[topic]; !exists {
		fmt.Printf("T[%s] has no listeners to emit to", topic)
		e.mu.Unlock()
		return
	}
	handlers := e.listeners[topic]
	e.mu.Unlock()
	handlers.mu.Lock()

	// fmt.Printf("T[%s] emitting with %d args\n", topic, len(args))

	// Construct input args and make sure the types in args match
	// the types specified by the handlers:
	inArgs := make([]reflect.Value, len(args))

	if len(args) != len(handlers.in) {
		panicF("T[%s] has %d input args; got %d", topic, len(handlers.in), len(args))
	}

	for i, arg := range args {
		t := reflect.TypeOf(arg)
		if t.Kind() != handlers.in[i].Kind() {
			panicF("T[%s] invalid argument at position %d. Expected %s; got %s", topic, i, handlers.in[i].Kind(), t.Kind())
		}

		inArgs[i] = reflect.ValueOf(arg)
	}

	// Gather the handlers to be called, removing any that are registered as "once":
	funcs := handlers.grabFuncs()

	// Unlock mutex - we're done mutating this object, and the callbacks
	// may need to use it.
	handlers.mu.Unlock()

	// Call each function with the input args
	if doSync {
		callFuncsSync(funcs, inArgs)
	} else {
		callFuncsAsync(funcs, inArgs)
	}
}

func (h *Handlers) grabFuncs() []reflect.Value {
	funcs := make([]reflect.Value, len(h.funcs))
	notRemoved := make([]HandlerFunc, 0)

	for i := 0; i < len(h.funcs); i++ {
		funcs[i] = h.funcs[i].f

		// Remove callback now that we're firing it
		if !h.funcs[i].once {
			notRemoved = append(notRemoved, h.funcs[i])
		}
	}

	h.funcs = notRemoved
	return funcs
}

func callFuncsSync(funcs []reflect.Value, inArgs []reflect.Value) {
	for _, f := range funcs {
		f.Call(inArgs)
	}
}

func callFuncsAsync(funcs []reflect.Value, inArgs []reflect.Value) {
	for _, f := range funcs {
		go f.Call(inArgs)
	}
}

func panicF(format string, a ...interface{}) {
	panic(fmt.Sprintf(format, a...))
}
