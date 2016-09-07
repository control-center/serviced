// Copyright 2016 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package registry

import (
	"path"
	"sync"

	"github.com/control-center/serviced/coordinator/client"
)

// StringMatcher is for matching applications
type StringMatcher interface {
	MatchString(string) bool
}

// Term describes a what applications it is looking for and then sends the
// response.
type Term struct {
	matcher StringMatcher
	send    chan<- string
}

// ImportListener passes matching endpoints to its respective receiver channel
type ImportListener struct {
	tenantID string
	terms    []Term
	apps     map[string]struct{}
	mu       *sync.Mutex
}

// NewImportListener instantiates a new listener for a given tenant id.
func NewImportListener(tenantID string) *ImportListener {
	return &ImportListener{
		tenantID: tenantID,
		apps:     make(map[string]struct{}),
	}
}

// AddTerm adds a search term to the listener and returns a channel receiever
// with the applicable matches.
func (l *ImportListener) AddTerm(matcher StringMatcher) <-chan string {
	send := make(chan string)
	t := Term{
		matcher: matcher,
		send:    send,
	}
	l.terms = append(l.terms, t)

	return send
}

// Run starts the export listener for the given tenant
func (l *ImportListener) Run(cancel <-chan struct{}, conn client.Connection) {
	logger := plog.WithField("tenantid", l.tenantID)

	pth := path.Join("/net/export", l.tenantID)

	done := make(chan struct{})
	defer func() { close(done) }()
	for {

		// set a watcher to alert when the path is available
		ok, ev, err := conn.ExistsW(pth, done)
		if err != nil {
			logger.WithError(err).Error("Could not listen for exports on tenant")
			return
		}

		var ch []string
		if ok {
			// the path is available, so set the listener on the change in the
			// number of children
			ch, ev, err = conn.ChildrenW(pth, done)
			if err == client.ErrNoNode {
				logger.Debug("Import was suddenly deleted, retrying")
				close(done)
				done = make(chan struct{})

				continue
			} else if err != nil {
				logger.WithError(err).Error("Could not listen for exports on tenant")
				return
			}
		}

		// for each new application found, send the matches to the respective
		// term channels.
		for _, app := range ch {
			if _, ok := l.apps[app]; !ok {
				for _, t := range l.terms {
					if t.matcher.MatchString(app) {
						select {
						case t.send <- app:
						case <-cancel:
							return
						}
					}
				}
				l.apps[app] = struct{}{}
			}
		}

		// wait for something to happen
		select {
		case <-ev:
		case <-cancel:
			return
		}

		close(done)
		done = make(chan struct{})
	}
}
