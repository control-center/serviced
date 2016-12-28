package web

import rest "github.com/zenoss/go-json-rest"

type EmergencyShutdownRequest struct {
	Operation int // 0 is emergency shutdown, 1 is clear emergency shutdown status
	TenantID  string
}

func restEmergencyShutdown(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	req := EmergencyShutdownRequest{}
	err := r.DecodeJsonPayload(&req)
	if err != nil {
		plog.WithError(err).Error("Could not decode json payload for emergency shutdown request")
		restBadRequest(w, err)
		return
	}
	err = ctx.getFacade().EmergencyShutdownRequest(req.TenantID, req.Operation)
	if err != nil {
		plog.WithError(err).Error("Facade could not process Emergency Shutdown Request")
		restBadRequest(w, err)
	}
	restSuccess(w)
}
