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

// Internal state needed by the Actor.
type Actor struct {
	In       chan Request
	Receiver reflect.Value
	Deferred bool
	Current  chan Response
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

// Defers responding to a particular call. Returns a Reply object that
// represents the reply. If an actor invokes this function, it promises
// to eventually call Send or Panic on the reply object.
func (r *Actor) Defer(function interface{}, args ...interface{}) {
	r.Deferred = true
	go r.runDeferred(Reply{Response: r.Current, Replied: false}, function, args)
}

func (r *Actor) runDeferred(reply Reply, function interface{}, args []interface{}) {
	valueArgs := make([]reflect.Value, len(args))
	for i := 0; i < len(args); i++ {
		valueArgs[i] = reflect.ValueOf(args[i])
	}
	reply.Send(r.Guard(reflect.ValueOf(function), valueArgs))
}

func (r *Actor) Guard(function reflect.Value, args []reflect.Value) (response Response) {
	defer func() {
		if e := recover(); e != nil {
			response = Response{result: nil, err: e, panicked: true}
		}
	}()

	result := function.Call(args)
	response = Response{result: result, err: nil, panicked: false}
	return
}

func (r *Actor) processOneRequest(request Request) {
	r.Deferred = false
	r.Current = request.response
	response := r.Guard(request.function, request.args)
	if !r.Deferred {
		request.response <- response
	}
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

type Reply struct {
	Response chan Response
	Replied  bool
}

// Indicates that a message has finished processing. Sends a reply to the
// sender indicating this.
func (r *Reply) Send(response Response) {
	if r.Replied {
		panic("Send/Panic called twice!")
	}

	r.Replied = true

	r.Response <- response
}
