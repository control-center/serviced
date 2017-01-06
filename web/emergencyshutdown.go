package web

import (
	"github.com/control-center/serviced/dao"
	rest "github.com/zenoss/go-json-rest"
)

type EmergencyShutdownRequest struct {
	TenantID string
}

func restEmergencyShutdown(w *rest.ResponseWriter, r *rest.Request, ctx *requestContext) {
	req := EmergencyShutdownRequest{}
	err := r.DecodeJsonPayload(&req)
	if err != nil {
		plog.WithError(err).Error("Could not decode json payload for emergency shutdown request")
		restBadRequest(w, err)
		return
	}
	_, err = ctx.getFacade().EmergencyStopService(ctx.getDatastoreContext(), dao.ScheduleServiceRequest{req.TenantID, true, false})
	if err != nil {
		plog.WithError(err).Error("Facade could not process Emergency Shutdown Request")
		restBadRequest(w, err)
	}
	restSuccess(w)
}
