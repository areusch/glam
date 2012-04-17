package glam

import(
	"container/list"
)

type MessageQueue struct {
	Q *list.List
	Limit int
	In chan Request
	Out chan Request
}

func NewMessageQueue(limit int) *MessageQueue {
	q := new(MessageQueue)
	q.Q = list.New()
	q.Limit = limit
	q.In = make(chan Request)
	q.Out = make(chan Request)
	go q.Run()
	return q
}

func (q *MessageQueue) processIn(msg Request) bool {
	if msg.Function.IsNil() {
		q.drain()
		close(q.In)
		close(q.Out)
		return false
	}
	q.Q.PushBack(msg)
	return true
}

func (q *MessageQueue) doIn() bool {
	return q.processIn(<- q.In)
}

func (q *MessageQueue) doInOut() bool {
	select {
	case msg := <- q.In:
		return q.processIn(msg)
	case q.Out <- q.Q.Front().Value.(Request):
		q.Q.Remove(q.Q.Front())
	}
	return true
}

func (q *MessageQueue) doOut() {
	q.Out <- q.Q.Front().Value.(Request)
	q.Q.Remove(q.Q.Front())
}

func (q *MessageQueue) Run() {
	for {
		if q.Q.Len() == 0 {
			if !q.doIn() {
				return
			}
		} else if q.Q.Len() < q.Limit {
			if !q.doInOut() {
				return
			}
		} else {
			q.doOut()
		}
	}
}

func (q *MessageQueue) drain() {
	for {
		select {
		case <- q.In:
			continue
		default:
			return
		}
	}
}
