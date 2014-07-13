// Copyright 2014, The Serviced Authors. All rights reserved.
// Use of this source code is governed by a
// license that can be found in the LICENSE file.

package web

import (
	"github.com/zenoss/glog"
	"github.com/zenoss/go-json-rest"
	"github.com/zenoss/serviced/domain/event"

	"net/url"
)

//restGetEvents gets all events. Response is map[event-id]event.Event
func restGetEvents(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	response := make(map[string]*event.Event)
	client, err := ctx.getMasterClient()
	if err != nil {
		restServerError(w)
		return
	}

	events, err := client.GetEvents()
	if err != nil {
		glog.Errorf("Could not get events: %v", err)
		restServerError(w)
		return
	}
	glog.V(2).Infof("Returning %d events", len(events))
	for _, event := range events {
		response[event.ID] = event
	}

	w.WriteJson(&response)
}

//restGetEvent retrieves a event. Response is Event
func restGetEvent(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	eventID, err := url.QueryUnescape(r.PathParam("eventID"))
	if err != nil {
		restBadRequest(w)
		return
	}

	client, err := ctx.getMasterClient()
	if err != nil {
		restServerError(w)
		return
	}

	event, err := client.GetEvent(eventID)
	if err != nil {
		glog.Error("Could not get event: ", err)
		restServerError(w)
		return
	}

	glog.V(4).Infof("restGetEvent: id %s, event %#v", eventID, event)
	w.WriteJson(&event)
}

//restRemoveEvent removes a event using event-id
func restRemoveEvent(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	eventID, err := url.QueryUnescape(r.PathParam("eventID"))
	if err != nil {
		restBadRequest(w)
		return
	}

	masterClient, err := ctx.getMasterClient()
	if err != nil {
		glog.Errorf("Unable to get client: %v", err)
		restServerError(w)
		return
	}
	err = masterClient.RemoveEvent(eventID)
	if err != nil {
		glog.Errorf("Could not remove event: %v", err)
		restServerError(w)
		return
	}
	glog.V(0).Info("Removed event ", eventID)
	w.WriteJson(&simpleResponse{"Removed event", eventsLinks(eventID)})
}
