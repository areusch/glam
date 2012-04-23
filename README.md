GLAM
====
(GoLang Actor Model)

**Disclaimer**: This is *all* experimental. I'm in no way, shape, or form an expert on Actor Model Concurrency.

So Go kind of sucks for concurrently accessing shared data structures. Sure, its primitives are powerful and can lend themselves to a high performance system. For the 99.9% of cases where performance isn't mission-crictical, it could really use a simple way to do the following:

```go
// Phonebook shared among many threads.
type Phonebook struct {
     book map[string]int
}

func (b Phonebook) Lookup(name string) (int, bool) {
     return book[name]  // Thread safety! Yay!
}

func (b *Phonebook) Add(name string, number int) {
     book[name] = number  // May I please have a segfault?
}

// From some webserver, e.g. many concurent goroutines:
func HandleAddRequest(name string, number int) {
  book.Add(name, number)
}
```

Turns out that using channels requires an absurd amount of boilerplate. Here's an attempt to reduce that.

Usage
=====

```go
import "github.com/areusch/glam"

type Phonebook struct {
     glam.Actor
     book map[string]int
}

func (b *Phonebook) Add(name string, number int) {
     book[name] = number
}

func (b *Phonebook) Lookup(name string) (int, bool) {
     return book[name]
}

func (b *Phonebook) LongRunningImportFromFile(reader io.Reader) int {
     // Importing a phonebook might take a long time and block on I/O. We can still
     // handle this message in the context of the actor, but run it in a deferred
     // fashion. Defer() launches a new goroutine and executes DoImport in the new
     // routine. When this function returns, no response is sent to the caller.
     b.Defer((*B).DoImport, b, reader)
}

// This function is executed in a new goroutine. If it needs to manipulate any state on
// b, it should invoke a method to do so using b.Call.
func (b *Phonebook) DoImport(r io.Reader, reply glam.Reply) int {
     numOk := 0
     for entry, err := ReadOneEntry(r); err == nil; entry, err = ReadOneEntry(r) {
          // From deferred functions, you can still send messages to the original actor.
          if ok := b.Call((*Phonebook).Add, entry["name"], entry["number"])[0].Bool(); ok {
               numImported++
          }
     }

     return numOk
}

func main() {
     // herp derp
     book := Phonebook{glam.Actor{}, make(map[string]int)}
     book.StartActor(&book)  // Call this before calling "Call"
     book.Call((*Phonebook).Lookup, "Jane")[0].Int()
}
```

Performance
===========

The tests include a benchmark. On my 2008 MBP: (note these are preliminary, need to look into the channel one)

```
glam_test.BenchmarkActor	  500000	      5577 ns/op
glam_test.BenchmarkChannel	 1000000	      1042 ns/op
glam_test.BenchmarkDeferred	  200000	      8022 ns/op
```

So it's about 5.5x worse for trivial functions. No testing has been done against large numbers of arguments or situations where function calls may block.
