package shell

type _empty struct{}
type semaphore chan _empty

// acquire n resources
func (s semaphore) P(n int) {
	e := _empty{}
	for i := 0; i < n; i++ {
		s <- e
	}
}

// release n resources
func (s semaphore) V(n int) {
	for i := 0; i < n; i++ {
		<-s
	}
}

/* mutexes */
func (s semaphore) Lock() {
	s.P(1)
}

func (s semaphore) Unlock() {
	s.V(1)
}

/* signal-wait */
func (s semaphore) Signal() {
	s.V(1)
}

func (s semaphore) Wait(n int) {
	s.P(n)
}
