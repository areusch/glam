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
     return book[name]
}

func (b *Phonebook) Add(name string, number int) {
     book[name] = number
}

// From some webserver, e.g. many concurent goroutines:
book.Lookup("Jane")
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

func (me Phonebook) Lookup(name string) (int, bool) {
     return true
}

func main() {
     // herp derp
     book := Phonebook{glam.Actor{}, make(map[string]int)}
     book.StartActor(book)
     book.Call(book.Lookup, "Jane")[0].Int()
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
