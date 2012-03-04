GLAM
====
(GoLang Actor Model)

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
     // fashion. We call Defer(), launch a new goroutine and return from this message
     // handler. The new goroutine is responsible for sending the reply.
     go b.DoImport(reader, b.Defer())
}

// This function is ha
func (b *Phonebook) DoImport(r io.Reader, reply glam.Reply) {
     numOk := 0
     for entry, err := ReadOneEntry(r); err == nil; entry, err = ReadOneEntry(r) {
          // From deferred functions, you can still send messages to the original actor.
          if ok := b.Call((*Phonebook).Add, entry["name"], entry["number"])[0].Bool(); ok {
               numImported++
          }
     }

     reply.Send(numOk)
}

func main() {
     // herp derp
     book := Phonebook{glam.Actor{}, make(map[string]int)}
     book.StartActor(*book)  // Call this before calling "Call"
     book.Call((*Phonebook).Lookup, "Jane")[0].Int()
}
```

Performance
===========

The tests include a benchmark. On my 2008 MBP:

```
glam_test.BenchmarkActor	 1000000	      5256 ns/op
glam_test.BenchmarkChannel	 1000000	      2121 ns/op
```

So it's about 2.5x worse for trivial functions. No testing has been done against large numbers of arguments or situations where function calls may block.
