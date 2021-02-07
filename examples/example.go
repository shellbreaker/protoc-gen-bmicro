package main

import (
	"context"
	"github.com/astaxie/beego"
	"github.com/micro/go-micro/v2"
	"github.com/shellbreaker/protoc-gen-bmicro/examples/pb"
	"github.com/shellbreaker/protoc-gen-bmicro/gateway"
	gwRate "github.com/shellbreaker/protoc-gen-bmicro/gateway/rate"
	"time"
)

type E struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func (e *E) Set(c int, m string) {
	e.ErrCode, e.ErrMsg = c, m
}

func main() {
	rpc := micro.NewService(micro.Name("go.rpc.example"))
	rpc.Init(
		micro.AfterStart(func() error {
			cli := pb.NewRestFulService("go.rpc.example", rpc.Client())

			pb.RegisterRestFulGateway(cli,
				gateway.SetCustomError(new(E)),
				gateway.SetRateControl(
					gwRate.NewMemoryAdapter(),
					gwRate.NewRateLimiting("/api/v1/example", "get", 500*time.Millisecond),
				),
			)

			go beego.Run()
			return nil
		}),
	)

	if err := pb.RegisterRestFulHandler(rpc.Server(), new(REST)); err != nil {
		panic(err.Error())
	}
	if err := rpc.Run(); err != nil {
		panic(err.Error())
	}
}

type REST struct {
}

func (r *REST) Get(ctx context.Context, req *pb.Request, res *pb.Response) (err error) {
	err = req.Validate()
	if err == nil {
		res.Msg = "success"
	}
	return
}

func (r *REST) Post(ctx context.Context, req *pb.Request, res *pb.Response) error {
	return nil
}
