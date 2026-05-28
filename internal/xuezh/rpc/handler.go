package rpc

import (
	"net/http"

	"github.com/joshp123/xuezh/api/xuezh/v1/xuezhv1connect"
	"github.com/joshp123/xuezh/internal/xuezh/service"
)

type Handler struct {
	xuezhv1connect.UnimplementedXuezhServiceHandler
	app service.App
}

func NewHandler(app service.App) (string, http.Handler) {
	return xuezhv1connect.NewXuezhServiceHandler(&Handler{app: app})
}
