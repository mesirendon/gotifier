package gotifier

import (
	"fmt"
	"sync"
	"testing"
)

type Packet struct {
	txt int
}

type Router struct {
	queue    chan Packet
	notifier Notifier
}

type RouterConsumer interface {
	Enqueued(*Router, Packet)
	Forwarded(*Router, Packet)
	Dropped(*Router, Packet)
}

func (r *Router) Notify(c RouterConsumer) {
	r.notifier.Notify(c)
}

func (r *Router) StopNotify(c RouterConsumer) {
	r.notifier.StopNotify(c)
}

func (r *Router) notifyAll(notify func(c RouterConsumer)) {
	r.notifier.NotifyAll(func(c Consumer) {
		notify(c.(RouterConsumer))
	})
}

func (r *Router) Receive(p Packet) {
	select {
	case r.queue <- p:
		r.notifyAll(func(c RouterConsumer) {
			c.Enqueued(r, p)
		})
	default:
		r.notifyAll(func(c RouterConsumer) {
			c.Dropped(r, p)
		})
	}
}

func (r *Router) Forward() {
	p := <-r.queue
	r.notifyAll(func(c RouterConsumer) {
		c.Forwarded(r, p)
	})
}

type Metrics struct {
	enqueued  int
	forwarded int
	dropped   int
	received  chan Packet
	sync.Mutex
}

func (m *Metrics) Enqueued(r *Router, p Packet) {
	m.Lock()
	m.enqueued++
	m.Unlock()
	if m.received != nil {
		m.received <- p
	}
}

func (m *Metrics) Forwarded(*Router, Packet) {
	m.Lock()
	m.forwarded++
	m.Unlock()
	if m.received != nil {
		m.received <- Packet{}
	}
}

func (m *Metrics) Dropped(*Router, Packet) {
	m.Lock()
	m.dropped++
	m.Unlock()
	if m.received != nil {
		m.received <- Packet{}
	}
}

func (m *Metrics) String() string {
	m.Lock()
	defer m.Unlock()
	return fmt.Sprintf(
		"%d enqueued | %d forwarded | %d queued | %d dropped",
		m.enqueued, m.forwarded, m.enqueued-m.forwarded, m.dropped,
	)
}

func TestNotifies(t *testing.T) {
	m := Metrics{received: make(chan Packet)}
	r := Router{queue: make(chan Packet, 10)}
	r.Notify(&m)

	for i := 0; i < 10; i++ {
		r.Receive(Packet{
			txt: i,
		})
		<-m.received

		if m.enqueued != (i + 1) {
			t.Error("Not notifying correctly", m.enqueued, i+1)
		}
	}

	for i := 0; i < 10; i++ {
		r.Receive(Packet{})
		<-m.received

		if m.enqueued != 10 {
			t.Error("Not notifying correctly", m.enqueued, 10)
		}

		if m.dropped != (i + 1) {
			t.Error("Not notifying correctly", m.dropped, i+1)
		}
	}

	t.Log("m", m.String())
}

func TestStopsNotifying(t *testing.T) {
	m := Metrics{received: make(chan Packet)}
	r := Router{queue: make(chan Packet, 10)}
	r.Notify(&m)

	for i := 0; i < 5; i++ {
		r.Receive(Packet{})
		<-m.received

		if m.enqueued != (1 + i) {
			t.Error("not notifying correctly")
		}
	}

	r.StopNotify(&m)

	for i := 0; i < 5; i++ {
		r.Receive(Packet{})

		select {
		case <-m.received:
			t.Error("did not stop notifying")
		default:
		}

		if m.enqueued != 5 {
			t.Error("did not stop notifying")
		}
	}
}

func TestThreadsafe(t *testing.T) {
	N := 1000
	r := Router{queue: make(chan Packet, 10)}
	m1 := Metrics{received: make(chan Packet)}
	m2 := Metrics{received: make(chan Packet)}
	m3 := Metrics{received: make(chan Packet)}

	r.Notify(&m1)
	r.Notify(&m2)
	r.Notify(&m3)

	var n int
	var wg sync.WaitGroup

	for i := 0; i < N; i++ {
		n++
		wg.Add(1)

		go func() {
			defer wg.Done()
			r.Receive(Packet{})
		}()

		if i%3 == 0 {
			n++
			wg.Add(1)

			go func() {
				defer wg.Done()
				r.Forward()
			}()
		}
	}

	for i := 0; i < (n * 3); i++ {
		select {
		case <-m1.received:
		case <-m2.received:
		case <-m3.received:
		}
	}

	t.Log("m1", m1.String())
	t.Log("m2", m2.String())
	t.Log("m3", m3.String())

	wg.Wait()

	t.Log("m1", m1.String())
	t.Log("m2", m2.String())
	t.Log("m3", m3.String())

	if m1.String() != m2.String() || m2.String() != m3.String() {
		t.Error("counts disagree")
	}
}
