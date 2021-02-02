package main

import (
	"context"
	"errors"
	"github.com/astaxie/beego"
	//_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-grpc-gateway"
	//_ "github.com/grpc-ecosystem/grpc-gateway/v2/protoc-gen-openapiv2"
	//_ "google.golang.org/grpc/cmd/protoc-gen-go-grpc"
	//_ "google.golang.org/protobuf/cmd/protoc-gen-go"
	"github.com/micro/go-micro/v2"
	gateway "protoc-gen-bmicro/bmicro"
)

type E struct {
	ErrCode int    `json:"errcode"`
	ErrMsg  string `json:"errmsg"`
}

func (e *E) Set(c int, m string) {
	e.ErrCode, e.ErrMsg = c, m
}

func main() {

	rpc := micro.NewService(micro.Name("go.rpc.test"))
	rpc.Init(
		micro.AfterStart(func() error {
			cli := gateway.NewRESTService("go.rpc.test", rpc.Client())

			gateway.RegisterRESTGateway(cli, gateway.SetCustomError(new(E)))

			go beego.Run()
			return nil
		}),
	)

	if err := gateway.RegisterRESTHandler(rpc.Server(), new(REST)); err != nil {
		panic(err.Error())
	}
	if err := rpc.Run(); err != nil {
		panic(err.Error())
	}
}

type REST struct {
}

func (r *REST) Get(ctx context.Context, req *gateway.Request, res *gateway.Response) (err error) {

	if req.Query == "1" {
		res.Code, res.Msg = 1, "err"
		return
	} else if req.Query == "2" {
		return errors.New("throw new error")
	}

	res.Msg = req.Query
	return
}

func (r *REST) Post(ctx context.Context, req *gateway.Request, res *gateway.Response) error {
	res.Msg = req.Body
	return nil
}
