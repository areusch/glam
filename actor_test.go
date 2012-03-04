package glam_test;

import (
	"glam"
	"testing"
)

type A struct {
	x int
	y int
	in chan GetXRequest
	glam.Actor
}

type B interface {
	Tricks() int
}

type GetXRequest struct {
	x int
	out chan GetXResponse
}

type GetXResponse struct {
	x int
	err interface{}
}

func (a A) GoX(x int) int {
	out := make(chan GetXResponse)
	a.in <- GetXRequest{x, out}
	return (<- out).x
}

func (a A) ProcessGetX() {
	for {
		request := <-a.in
		request.out <- GetXResponse{a.GetX(request.x), nil}
	}
}

func (a A) GetX(x int) int {
	return a.x + x
}

func (a A) DoPanic() int {
	panic(a.y)
}

func (a *A) Tricks() int {
	go a.LongTricks(a.x, a.Defer())
	return a.x
}

func (a A) LongTricks(x int, reply glam.Reply) {
	reply.Send(x + 5)
}

func TestGetX(t *testing.T) {
	a := A{2, 3, nil, glam.Actor{}}
	a.StartActor(a)

	if x := a.Call(A.GetX, 4)[0].Int(); x != 6 {
		t.Errorf("Expected x = %v, actual %v\n", 6, x)
	}
}

func TestPanic(t *testing.T) {
	a := A{2, 3, nil, glam.Actor{}}
	a.StartActor(a)

	defer func() {
		if e := recover(); e != 3 {
			t.Errorf("Expected panic(3), actual %v\n", e)
		}
	}()

	a.Call(A.DoPanic)
}

func TestDefer(t *testing.T) {
	a := A{3, 4, nil, glam.Actor{}}
	a.StartActor(&a)
	if val := a.Call((*A).Tricks)[0].Int(); val != 8 {
		t.Errorf("Expected returning x+5, actual %v\n", val)
	}
}

func BenchmarkActor(b *testing.B) {
	b.StopTimer()
	a := A{5, 10, nil, glam.Actor{}}
	a.StartActor(a)
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		a.Call(A.GetX, 3)
	}
}

func BenchmarkChannel(b *testing.B) {
	b.StopTimer()
	a := A{5, 10, make(chan GetXRequest), glam.Actor{}}
	go a.ProcessGetX()
	b.StartTimer()
	for i := 0; i < b.N; i++ {
		a.GoX(3)
	}
}
