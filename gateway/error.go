package gateway

type Error interface {
	Set(int, string)
}

type ErrorBase struct {
	Code int    `json:"code"`
	Msg  string `json:"msg"`
}

func (e *ErrorBase) Set(c int, m string) {
	e.Code, e.Msg = c, m
}
