package leader

import (
	"github.com/zenoss/go-coordinator"
)

// Leader is used by a LeaderSelector for notification
// of a Leadership event.
type Leader interface {

	// The ID to store for the leader
	Id() string

	// TakeLead is called when the listener has been elected the leader.
	// It should return when it is finished with it's leadership duties.
	TakeLead(coordinator.Coordinator) error

	// InterruptLead is called by the Leader Selector in case the selector is
	// closed while the leader is in TakeLead().
	InterruptLead() error
}

// Selector is an abstraction to select a "leader" amongst multiple contenders in a
// group of applications connected to a Zookeeper cluster. If a group of N processes
// contends for leadership, one will be assigned leader until it releases leadership
// at which time another one from the group will be chosen.
type Selector struct {
	coordinator   coordinator.Coordinator
	leaderPath    string          // The path the leader is elected at.
	leader        Leader          // The leader this selector is managing
	hasLeadership uint32          // Non-zero if the selector's leader is the current leader.
	closing       chan chan error // closing channel
}

// HasLeadership() returns true if the select's leader has taken the lead.
func (s *Selector) HasLeadership() bool {
	isLeader := atomic.LoadUint32(&s.hasLeadership)
	return isLeader > 0
}


func (s *Selector)
// Close() shuts down the selector
func (s *Selector) Close() error {
	errc := make(chan error)
	s.closing <- errc
	return <-errc
}

func (s *Selector) loop() {
	var err error
	var leadEnded chan error
	for {

		// create the leaderPath

		select {

		// handle case where leader has exited TakeLead
		case err = <-leadEnded:

		// handle the shutdown case
		case errc := <-s.closing:
			errc <- err
			return
		}
	}
}

// NewSelector creates a new Leader Selector. It will call the leader's TakeLead method
// when it is elected leader.
func NewSelector(coordinator coordinator.Coordinator, leaderPath string, leader Leader) (selector *Selector, err error) {
	selector = &Selector{
		coordinator: coordinator,
		leaderPath:  leaderPath,
		leader:      leader,
	}
	go s.loop()
	return selector, nil
}
