package gateway

import (
	"github.com/astaxie/beego"
	"github.com/astaxie/beego/context"
	jsoniter "github.com/json-iterator/go"
	gwRate "github.com/shellbreaker/protoc-gen-bmicro/gateway/rate"
	"reflect"
	"strings"
)

var json = jsoniter.ConfigCompatibleWithStandardLibrary

type settings struct {
	CustomError    Error
	RateLimits     []gwRate.FingerPrint
	RateCtlAdapter gwRate.Adapter
}

func (s *settings) isRateCtlOn() bool {
	return s.RateCtlAdapter != nil && len(s.RateLimits) > 0
}

func (s *settings) InsertRateLimiting() {
	if s.isRateCtlOn() {
		for _, r := range s.RateLimits {
			beego.InsertFilter(r.Path(), beego.BeforeExec, func(c *context.Context) {
				if strings.ToLower(c.Input.Method()) == r.MethodLower() {
					r.Ip(c.Input.Header("X-Forwarded-For"))
					if lim := gwRate.Limiter(s.RateCtlAdapter, r); !lim.Allow() {
						content := s.NewError()
						content.Set(10403, "Request was denied due to throttling-control")
						j, _ := json.Marshal(content)
						c.WriteString(string(j))
					}
				}
			})
		}
	}
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

func SetRateControl(adptr gwRate.Adapter, fps ...gwRate.FingerPrint) Option {
	return Option{func(s *settings) {
		s.RateLimits, s.RateCtlAdapter = fps, adptr
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
