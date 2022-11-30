package gotifier

import "sync"

type Consumer interface{}

type Notifier struct {
	mut       sync.RWMutex
	consumers map[Consumer]struct{}
}

func (n *Notifier) Notify(c Consumer) {
	n.mut.Lock()
	if n.consumers == nil {
		n.consumers = make(map[Consumer]struct{})
	}
	n.consumers[c] = struct{}{}
	n.mut.Unlock()
}

func (n *Notifier) StopNotify(c Consumer) {
	n.mut.Lock()
	if n.consumers != nil {
		delete(n.consumers, c)
	}
	n.mut.Unlock()
}

func (n *Notifier) NotifyAll(notify func(Consumer)) {
	n.mut.Lock()
	defer n.mut.Unlock()

	if n.consumers == nil {
		return
	}

	for consumer := range n.consumers {
		go notify(consumer)
	}
}
