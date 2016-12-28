package web

import (
	"github.com/control-center/serviced/dao"
	rest "github.com/zenoss/go-json-rest"
)

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
	stateID := "stop"
	if req.Operation != 0 {
		stateID = "go"
	}
	daoReq := dao.ServiceStateRequest{
		ServiceID:      req.TenantID,
		ServiceStateID: stateID,
	}
	n, err := ctx.getFacade().EmergencyStopService(ctx, daoReq)
	if err != nil {
		plog.WithError(err).Error("Facade could not process Emergency Shutdown Request")
		restBadRequest(w, err)
		return
	}
	plog.Infof("Scheduled %d services", n)
	restSuccess(w)
}
