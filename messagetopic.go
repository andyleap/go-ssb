package ssb

import "sync"

type MessageTopic struct {
	lock  sync.Mutex
	recps map[chan *SignedMessage]bool
	Send  chan *SignedMessage
}

func NewMessageTopic() *MessageTopic {
	mt := &MessageTopic{Send: make(chan *SignedMessage, 10), recps: map[chan *SignedMessage]bool{}}
	go mt.process()
	return mt
}

func (mt *MessageTopic) Close() {
	close(mt.Send)
}

func (mt *MessageTopic) process() {
	for m := range mt.Send {
		func() {
			mt.lock.Lock()
			defer mt.lock.Unlock()
			for recp, strict := range mt.recps {
				if strict {
					recp <- m
				} else {
					select {
					case recp <- m:
					default:
						delete(mt.recps, recp)
						close(recp)
					}
				}
			}
		}()

	}
	mt.lock.Lock()
	defer mt.lock.Unlock()
	for recp := range mt.recps {
		delete(mt.recps, recp)
		close(recp)
	}
}

func (mt *MessageTopic) Register(recp chan *SignedMessage, strict bool) chan *SignedMessage {
	mt.lock.Lock()
	defer mt.lock.Unlock()
	if recp == nil {
		recp = make(chan *SignedMessage, 1)
	}
	mt.recps[recp] = strict
	return recp
}

func (mt *MessageTopic) Unregister(recp chan *SignedMessage) {
	mt.lock.Lock()
	defer mt.lock.Unlock()

	if _, ok := mt.recps[recp]; ok {
		delete(mt.recps, recp)
		close(recp)
	}
}
