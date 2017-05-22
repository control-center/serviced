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

/*
	Package audit implements audit logging.

	This package provides an implementation of the Logger interface that can
	be used to perform audit logging.

	Entries

	The default logger implementation will create entries in the log file with the following fields.

		time: 	 The time that the action took place.
		level: 	 The log level for the action.  Warnings will be for failed actions, and info for successful actions.
		action:	 The action being performed (e.g. add, remove, update, start, stop, etc).
		type:    Entity being changed if applicable (e.g. resourcepool, host, service, etc).
		id:      Id of entity being changed if applicable.
		msg:     User friending message about what action took place.
		success: Either true or false.
		user: 	 The user performing the action.  This will default to "system" if no user is provided.

	An entry for a successful attempt to add a resouce pool.

		time="2017-05-11T19:41:10Z" level=info msg="Adding Resource Pool Swimming" action=add success=true user=system type=resourcepool id=Swimming

	An entry for a failed attempt to add a resouce pool.

		time="2017-05-11T19:41:10Z" level=warning msg="Adding Resource Pool Swimming" action=add success=false user=system type=resourcepool id=Swimming

	API

	The Logger interface provides a fluent API for creating log entries.  A new Logger can be retrieved by calling the NewLogger method.

		var auditLogger = audit.NewLogger()

	The default implementation that is returned is a wrapped logri Logger.  Logri is a package owned by Zenoss that adds additional
	functionality to Loggers from the third party package, logrus.

	To log an entry , a message along with relavant contextual information must be set on the audit logger through the use of the "Message" method.  This will set the "msg" and "user" fields:

		auditLogger = auditLogger.Message(ctx, "Adding Resource Pool")

	The Message method will take an implementation of the "datastore.context" interface.  Currently the audit logger only uses user field.  In the future, additional contextual information maybe utilized.

	The "action field is set through the "Action" method.  Some constants are provided in this package to normalized the values.  Examples are "add", "update",
	"delete", "start", "stop", etc.  If addition actions need to be added, new constants should be added to this package.

		auditLogger = auditLogger.Action(audit.Add)

	The interface, "Entity", is added to the "datastore" package to provide a convenient method for encapsulating a domain object's ID and type. The "type" and "ID" of an audit log entry can be set via the "Entity" method. "Entity" is the preferred method for setting an audit log entry's "id" and "type", but if an implementation of the "Entity" is not available, the methods "ID" and "Type" are available.

		auditLogger = auditLogger.Entity(entity)

	The type of entity being modified is set through the "Type" method.  To get normalized type values for a domain object, use the package method "GetType" for a given domain object.

		auditLogger = auditLogger.Type(pool.GetType())

	To set the "id" field, use the "ID" method.

		auditLogger = auditLogger.ID("PoolID")

	The API also provides a generic "WithFields" method.  If a custom, one-off field needs to be added to an entry, this method can be used.

		auditLogger = auditLogger.WithField("FieldName", "ItsValue")

	There are a number of ways that can be used to trigger logging which will also signal success or failure.  The signal success, use the "Success"
	method.

		auditLogger.Success()

	The "Success" method will set the "success" field to "true" and the "level" to "info".  This will also signal the audit logger to log.

	To log a failed action, use the "Failure" method.  This will set "success" to "false" and the "level" to "warning".

		auditLogger.Failure()

	There are also a couple of methods that have been added for convenience.  These are the "SucceededIf" and "Error" methods.  The "SucceededIf"
	method takes a boolean.  If passed in true, it will log that the action was successful in the same way that the "Success" method does.  If
	passed in false, it will log failure in similar fashion to the "Failure" message.

		auditLogger.SucceededIf(returnedValue == "EverythingOK")

	The "Error" method is convenient for working with methods that return only a single error type.  The convention is that a method will return
	a nil error if it was successful.  If something went wrong, the method will return a non-nil error.  The "Error" method will take an error and check to see
	if it is nil.  If it is, the audit logger  will then log success. If it is not nil, it will log failure.  The method also returns the error that
	was passed in.  This is convenient for wrapping calls.  For example, method could be defined that returns a single error.

		func DoSomething() error { ... }

	The could be code that we want to audit where that method is used.

		err := DoSomething()
		if err != nil {
			...
		}

	If we wanted to audit the success of failure of the "DoSomething" call, we could wrap it with a call to the "Error" method on the auditLogger.

		err := auditLogger.Error(DoSomething())
		if err != nil {
			...
		}

	In this case, the err variable will still have the same value as the previous call because the error returned by "DoSomething" is just passed through.
	Success or failure will be logged depending on the value of the error that is returned from the "DoSomething" method.

	The API is designed to be fluent so the method calls can be chained together.  For example, setting the fields and logging success can be done
	at the same time.  The following sets the context, message, action, id, type, and success:

		auditLogger.Message(ctx, "Adding Resource Pool").
		            Action(audit.Add).
		            Entity(entity).
					Success()


	This can also be seperated, into multiple groups of calls.  The values could be defined at the top of a method, and "Success" for "Failure" called later.

		func SomethingWeAuditing() {
			alog := auditLogger.Context(ctx).
						Message("Adding Resource Pool").
						Action(audit.Add).
						Entity(entity)


			// Do processing
			// ...

			if everythingOK {
				alog.Success()
			} else {
				alog.Failed()
			}

			return
		}


	Common Patterns

	Here is an example using the "SucceededIf" and "Failed" pattern to add audit logging to a method that adds resource pools.

		func (f *Facade) AddResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
			defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.AddResourcePool"))

			glog.Infof("Adding Resource Pool %s", entity.ID)

			alog := f.auditLogger.Message(ctx, "Adding Resource Pool").
				Action(audit.Add).Entity(entity)

			if err := f.DFSLock(ctx).LockWithTimeout("add resource pool", userLockTimeout); err != nil {
				glog.Warningf("Cannot add resource pool: %s", err)
				alog.Failure()
				return err
			}
			defer f.DFSLock(ctx).Unlock()

			err := f.addResourcePool(ctx, entity)

			alog.SuceededIf(err == nil)

			return err
		}

	Since this method returns just an error, the "Error" method can be used to make things more concise.

		func (f *Facade) AddResourcePool(ctx datastore.Context, entity *pool.ResourcePool) error {
			defer ctx.Metrics().Stop(ctx.Metrics().Start("Facade.AddResourcePool"))

			glog.Infof("Adding Resource Pool %s", entity.ID)

			alog := f.auditLogger.Message(ctx, "Adding Resource Pool").
				Action(audit.Add).Entity(entity)

			if err := f.DFSLock(ctx).LockWithTimeout("add resource pool", userLockTimeout); err != nil {
				glog.Warningf("Cannot add resource pool: %s", err)
				return alog.Error(err)
			}
			defer f.DFSLock(ctx).Unlock()

			return alog.Error(f.addResourcePool(ctx, entity))
		}

	In some methods, wrapping with the "Error" method might be more concise or easier to add.  However, there may be situations where the "Error" method is not adequate or a good fit.
	When the success or failure of an action is not dependent on an error, the "Succeeded", "SucceededIf", and "Failure" methods can then be used.
*/
package audit
