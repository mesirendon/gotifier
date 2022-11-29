package gotifier

import "sync"

type Consumer interface{}

type Notifier struct {
	mut      sync.RWMutex
	notifees map[Consumer]struct{}
}

func (n *Notifier) Notify(c Consumer) {
	n.mut.Lock()
	if n.notifees == nil {
		n.notifees = make(map[Consumer]struct{})
	}
	n.notifees[c] = struct{}{}
	n.mut.Unlock()
}

func (n *Notifier) StopNotify(c Consumer) {
	n.mut.Lock()
	if n.notifees != nil {
		delete(n.notifees, c)
	}
	n.mut.Unlock()
}

func (n *Notifier) NotifyAll(notify func(Consumer)) {
	n.mut.Lock()
	defer n.mut.Unlock()

	if n.notifees == nil {
		return
	}

	for notifee := range n.notifees {
		go notify(notifee)
	}
}
