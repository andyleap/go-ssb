package ssb

func (f *Feed) Follow(seq int, live bool, handler func(m *SignedMessage) error, done chan struct{}) error {
	for {
		f.SeqLock.Lock()
		if f.LatestSeq >= seq {
			f.SeqLock.Unlock()
			m := f.GetSeq(nil, seq)
			if m != nil {
				err := handler(m)
				if err != nil {
					return err
				}
			}
			seq++
		} else {
			if !live {
				f.SeqLock.Unlock()
				return nil
			}
			c := f.Topic.Register(nil, false)
			f.SeqLock.Unlock()
			for {
				select {
				case m := <-c:
					err := handler(m)
					if err != nil {
						return nil
					}
				case <-done:
					f.Topic.Unregister(c)
					return nil
				}
			}

		}
	}
}
