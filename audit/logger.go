// Copyright 2017 The Serviced Authors.
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

package audit

import (
	"strconv"

	"github.com/Sirupsen/logrus"
	"github.com/control-center/serviced/datastore"
	"github.com/control-center/serviced/logging"
	"github.com/zenoss/logri"
)

var plog = logging.PackageLogger()

// Logger is the interface for audit logging.  Any implementations for
// audit logging should implement this interface.
type Logger interface {

	// Set the action that we are auditing.
	Action(action string) Logger

	// Set the message that we are writing to the audit log.
	Message(ctx datastore.Context, message string) Logger

	// Set the type of entity being modified.
	Type(theType string) Logger

	// Set the id of the entity being modified.
	ID(id string) Logger

	// Set the type of entity being modified.
	Entity(entity datastore.Entity) Logger

	// Add an additional field to the entry.
	WithField(name string, value string) Logger

	// Add additional fields to the entry.
	WithFields(fields logrus.Fields) Logger

	// Log that the action succeeded.
	Succeeded()

	// Log that the action failed.
	Failed()

	// Log whether the action succeeded or failed based on the value passed in.
	SucceededIf(value bool)

	// Log whether the action succeeded or failed based on the error passed in.
	Error(err error) error
}

// NewLogger returns a default implmentation of the audit logger.  The "user" will default
// to "system" unless otherwise specified in the context.  This wraps a logri Logger,
// and will write to the location specified in the logger config.
func NewLogger() Logger {
	l := logri.GetLogger("audit")
	return &logger{loggeri: l}
}

type logger struct {
	entry   *logrus.Entry
	message string
	loggeri *logri.Logger
}

func (l *logger) Action(action string) Logger {
	return l.newLoggerWith("action", action)
}

func (l *logger) Message(ctx datastore.Context, message string) Logger {
	l.message = message
	return l.newLoggerWith("user", ctx.User())
}

func (l *logger) Type(theType string) Logger {
	return l.newLoggerWith("type", theType)
}

func (l *logger) ID(id string) Logger {
	return l.newLoggerWith("id", id)
}

func (l *logger) Entity(entity datastore.Entity) Logger {
	l.addField("id", entity.GetID())
	return l.newLoggerWith("type", entity.GetType())
}

func (l *logger) WithField(name string, value string) Logger {
	return l.newLoggerWith(name, value)
}

func (l *logger) newLoggerWith(name string, value string) Logger {
	l.addField(name, value)
	return &logger{
		entry:   l.entry,
		message: l.message,
		loggeri: l.loggeri,
	}
}

func (l *logger) addField(name string, value string) Logger {
	return l.addFields(logrus.Fields{name: value})
}

func (l *logger) WithFields(fields logrus.Fields) Logger {
	return l.addFields(fields)
}

func (l *logger) addFields(fields logrus.Fields) Logger {
	if l.entry != nil {
		l.entry = l.entry.WithFields(fields)
	} else {
		l.entry = l.loggeri.WithFields(fields)
		l.entry.Logger.Hooks = make(map[logrus.Level][]logrus.Hook)
	}
	return l
}

func (l *logger) Succeeded() {
	l.log(true)
}

func (l *logger) Failed() {
	l.log(false)
}

func (l *logger) SucceededIf(value bool) {
	l.log(value)
}

func (l *logger) Error(err error) error {
	l.log(err == nil)
	return err
}

func (l *logger) log(success bool) {
	l.addField("success", strconv.FormatBool(success))
	entry := l.entry
	if len(l.message) == 0 {
		pkgLogger := plog.WithFields(entry.Data)
		pkgLogger.Error("Attempting to audit log empty message")
	}
	if success {
		entry.Info(l.message)
	} else {
		entry.Warn(l.message)
	}
}
