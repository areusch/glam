package glam

import (
	"reflect"
)

// Represents a request to an actor's thread to invoke the given function with
// the given arguments.
type Request struct {
	function reflect.Value
	args     []reflect.Value
	response chan Response
}

// Represents the result of a function invocation.
type Response struct {
	result   []reflect.Value // The return value of the function.
	err      interface{}     // The value passed to panic, if it was called.
	panicked bool            // True if the invocation called panic.
}

// Internal state needed by
type Actor struct {
	In       chan Request
	Receiver reflect.Value
}

// Invoke function in the actor's own thread, passing args.
func (r Actor) Call(function interface{}, args ...interface{}) []reflect.Value {
	if r.In == nil {
		panic("Call StartActor before calling Do!")
	}

	// reflect.Call expects the arguments to be a slice of reflect.Values. We also
	// need to ensure that the 0th argument is the receiving struct.
	valuedArgs := make([]reflect.Value, len(args)+1)
	valuedArgs[0] = r.Receiver
	for i, x := range args {
		valuedArgs[i+1] = reflect.ValueOf(x)
	}

	out := make(chan Response, 0)
	r.In <- Request{reflect.ValueOf(function), valuedArgs, out}
	response := <-out

	if response.panicked {
		panic(response.err)
	}
	return response.result
}

func (r Actor) processOneRequest(request Request) {
	defer func() {
		if e := recover(); e != nil {
			request.response <- Response{err: e, panicked: true}
		}
	}()

	request.response <- Response{
		result:   request.function.Call(request.args),
		err:      nil,
		panicked: false}
}

// Start the internal goroutine that powers this actor. Call this function
// before calling Do on this object.
func (r *Actor) StartActor(receiver interface{}) {
	r.In = make(chan Request, 0)
	r.Receiver = reflect.ValueOf(receiver)
	go func() {
		for {
			request := <-r.In
			r.processOneRequest(request)
		}
	}()
}
