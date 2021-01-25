package gin

import (
	"encoding/json"
	"fmt"
	"github.com/gin-gonic/gin"
	"github.com/nic-chen/droplet"
	"github.com/nic-chen/droplet/data"
	"github.com/nic-chen/droplet/log"
	"github.com/nic-chen/droplet/middleware"
	"github.com/nic-chen/droplet/wrapper"
	"net/http"
)

func Wraps(handler droplet.Handler, opts ...wrapper.SetWrapOpt) func(*gin.Context) {
	return func(ctx *gin.Context) {
		opt := &wrapper.WrapOptBase{}
		for i := range opts {
			opts[i](opt)
		}

		dCtx := droplet.NewContext()
		dCtx.SetContext(ctx)

		ret, _ := droplet.NewPipe().
			Add(middleware.NewHttpInfoInjectorMiddleware(middleware.HttpInfoInjectorOption{
				ReqFunc: func() *http.Request {
					return ctx.Request
				},
			})).
			Add(middleware.NewRespReshapeMiddleware()).
			Add(middleware.NewHttpInputMiddleWare(middleware.HttpInputOption{
				PathParamsFunc: func(key string) string {
					return ctx.Param(key)
				},
				InputType:      opt.InputType,
				IsReadFromBody: opt.IsReadFromBody,
			})).
			Add(middleware.NewTrafficLogMiddleware(opt.TrafficLogOpt)).
			SetOrchestrator(opt.Orchestrator).
			Run(handler, droplet.InitContext(dCtx))

		switch ret.(type) {
		case *data.FileResponse:
			fr := ret.(*data.FileResponse)
			if fr.ContentType == "" {
				fr.ContentType = "application/octet-stream"
			}
			ctx.Writer.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=%s", fr.Name))
			ctx.Writer.Header().Set("Content-type", fr.ContentType)
			_, err := ctx.Writer.Write(fr.Content)
			if err != nil {
				log.Error("write response failed",
					"err", err)
			}
		case droplet.SpecCodeHttpResponse:
			resp := ret.(droplet.SpecCodeHttpResponse)
			bs, err := json.Marshal(resp)
			if err != nil {
				log.Error("marshal result failed",
					"err", err)
				return
			}
			ctx.Data(resp.GetStatusCode(), "application/json", bs)
		default:
			bs, err := json.Marshal(ret)
			if err != nil {
				log.Error("marshal result failed",
					"err", err)
				return
			}
			ctx.Data(http.StatusOK, "application/json", bs)
		}
	}
}
