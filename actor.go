package glam

import (
	"fmt"
	"reflect"
	"runtime/debug"
)

// Internal state needed by the Actor.
type Actor struct {
	Q *MessageQueue
	Receiver reflect.Value
	Deferred bool
	Current chan<- Response
}

const kActorQueueLength int = 1

// Synchronously invoke function in the actor's own thread, passing args. Returns the
// result of execution.
func (r *Actor) Call(function interface{}, args ...interface{}) []interface{} {
	out := make(chan Response, 0)
	r.Cast(out, function, args...)
	response := <-out

	return response.InterpretAsInterfaces()
}

// Internal method to verify that the given function can be invoked on the actor's
// receiver with the given args.
func (r *Actor) verifyCallSignature(function interface{}, args []interface{}) {
	typ := reflect.TypeOf(function)
	if typ.Kind() != reflect.Func {
		panic("Function is not a method!")
	}
	if typ.NumIn() < 1 {
		panic("Casted method has no receiver!")
	}
	if !r.Receiver.Type().AssignableTo(typ.In(0)) {
		panic(fmt.Sprintf(
			"Cannot assign receiver (of type %s) to %s", r.Receiver.Type(), typ.In(0)))
	}
	numNonReceiver := typ.NumIn() - 1
	if len(args) < numNonReceiver {
		panic(fmt.Sprintf(
			"Not enough arguments given (needed %d, got %d)", numNonReceiver, len(args)))
	}
	if len(args) > numNonReceiver && !typ.IsVariadic() {
		panic(fmt.Sprintf("Too many args for non-variadic function (needed %d, got %d)",
			numNonReceiver, len(args)))
	}
	for i := 1; i < typ.NumIn(); i++ {
		if argType := reflect.TypeOf(args[i-1]); !argType.AssignableTo(typ.In(i)) {
			panic(
				fmt.Sprintf("Cannot assign arg %d (%s -> %s)", i - 1, argType, typ.In(i)))
		}
	}
}

// Asynchronously request that the given function be invoked with the given args.
func (r *Actor) Cast(out chan<- Response, function interface{}, args ...interface{}) {
	r.verifyCallSignature(function, args)
	r.runInThread(out, r.Receiver, function, args...)
}

func (r *Actor) runInThread(out chan<- Response, receiver reflect.Value, function interface{}, args ...interface{}) {
	if r.Q == nil {
		panic("Call StartActor before sending it messages!")
	}

	// reflect.Call expects the arguments to be a slice of reflect.Values. We also
	// need to ensure that the 0th argument is the receiving struct.
	valuedArgs := make([]reflect.Value, len(args)+1)
	valuedArgs[0] = receiver
	for i, x := range args {
		valuedArgs[i+1] = reflect.ValueOf(x)
	}

	r.Q.In <- Request{reflect.ValueOf(function), valuedArgs, out}
}

// Defers responding to a particular call, but gives full control over the response
// to the calling function. Specifically, this function returns a Reply object that
// allows the caller to respond at any given point in the future. If an actor
// invokes this function, it promises to eventually call Send or Panic on the
// reply object. Failing to do this may cause program lockup or panic, since
// goroutines
//
// It is an error to call this function from anything but the message-processing
// goroutine.
func (r *Actor) DeferUnguarded() Reply {
	r.Deferred = true
	return Reply{Response: r.Current, Replied: false}
}

// Defers responding to a particular call, and invokes the given function in a
// new goroutine to finish processing the call. The new goroutine invokes the
// function in the same guarded style as the calling context.
//
// It is an error to call this function from anything but the message-processing
// goroutine.
func (r *Actor) Defer(function interface{}, args ...interface{}) {
	r.Deferred = true
	go r.runDeferred(Reply{Response: r.Current, Replied: false}, function, args...)
}

func (r *Actor) runDeferred(reply Reply, function interface{}, args ...interface{}) {
	valueArgs := make([]reflect.Value, len(args))
	for i := 0; i < len(args); i++ {
		valueArgs[i] = reflect.ValueOf(args[i])
	}
	reply.Send(guardedExec(reflect.ValueOf(function), valueArgs))
}

func guardedExec(function reflect.Value, args []reflect.Value) (response Response) {
	defer func() {
		if e := recover(); e != nil {
			response = ResponseImpl{result: nil, err: e, panicked: true, Stack: debug.Stack(), function: function, args: args}
		}
	}()

	result := function.Call(args)
	response = ResponseImpl{result: result, err: nil, panicked: false}
	return
}

func (r *Actor) processOneRequest(request Request) {
	r.Deferred = false
	r.Current = request.ReplyTo
	response := guardedExec(request.Function, request.Args)
	if request.ReplyTo != nil && !r.Deferred {
		request.ReplyTo <- response
	}
}

// Start the internal goroutine that powers this actor. Call this function
// before calling Do on this object.
func (r *Actor) StartActor(receiver interface{}) {
	r.Q = NewMessageQueue(kActorQueueLength)
	r.Receiver = reflect.ValueOf(receiver)
	go func() {
		for {
			request := <-r.Q.Out
			r.processOneRequest(request)
		}
	}()
}

type Reply struct {
	Response chan<- Response
	Replied  bool
}

// Indicates that a message has finished processing. Sends a reply to the
// sender indicating this.
func (r *Reply) Send(response Response) {
	if r.Replied {
		panic("Send/Panic called twice!")
	}

	r.Replied = true

	if r.Response != nil {
		r.Response <- response
	}
}
