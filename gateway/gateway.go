package gateway

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	jsoniter "github.com/json-iterator/go"
	"reflect"
	"strings"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type settings struct {
	CustomError Error
}

func (s *settings) NewError() Error {
	if s.CustomError != nil {
		rv := reflect.ValueOf(s.CustomError)
		if rv.Kind() == reflect.Ptr {
			rv = rv.Elem()
		}
		err, ok := reflect.New(rv.Type()).Interface().(Error)
		if ok {
			return err
		}
	}
	return new(ErrorBase)
}

type Option struct {
	fn func(s *settings)
}

type Options []Option

func (ops Options) Settings() *settings {
	gws := new(settings)
	for _, op := range ops {
		op.fn(gws)
	}
	return gws
}

func SetCustomError(e Error) Option {
	return Option{func(s *settings) {
		s.CustomError = e
	}}
}

func ParseParams(ctx *context.Context, obj interface{}) (err error) {
	if strings.Contains(strings.ToLower(ctx.Input.Header("content-type")), "json") {
		err = json.Unmarshal(ctx.Input.RequestBody, obj)
	} else {
		err = ctx.Request.ParseForm()
		if err == nil {
			err = beego.ParseForm(ctx.Request.Form, obj)
		}
	}
	return
}
