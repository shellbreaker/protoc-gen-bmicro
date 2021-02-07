package rate

import (
	"fmt"
	uuid "github.com/satori/go.uuid"
	"golang.org/x/time/rate"
	"strings"
	"sync"
	"time"
)

type FingerPrint interface {
	Rate() rate.Limit
	Hash() string
	Path() string
	MethodLower() string
	Ip(string)
}

type rateLimitingSetting struct {
	dependOnIp bool
	ip         string
	path       string
	method     string
	rate       rate.Limit
}

func (s *rateLimitingSetting) Hash() string {
	ip := ""
	if s.dependOnIp {
		ip = s.ip
	}
	return uuid.NewV5(uuid.NamespaceURL, fmt.Sprintf("method=%s&path=%s&ip=%s", s.method, s.path, ip)).String()
}

func (s *rateLimitingSetting) Rate() rate.Limit {
	return s.rate
}

func (s *rateLimitingSetting) Path() string {
	return s.path
}

func (s *rateLimitingSetting) MethodLower() string {
	return strings.ToLower(s.method)
}

func (s *rateLimitingSetting) Ip(input string) {
	s.ip = input
}

type Option struct {
	fn func(setting *rateLimitingSetting)
}

func Path(p string) Option {
	return Option{fn: func(setting *rateLimitingSetting) {
		setting.path = p
	}}
}

func Method(m string) Option {
	return Option{fn: func(setting *rateLimitingSetting) {
		setting.method = m
	}}
}

func Ip(ip string) Option {
	return Option{fn: func(setting *rateLimitingSetting) {
		setting.ip = ip
	}}
}

func DependOnIp() Option {
	return Option{fn: func(setting *rateLimitingSetting) {
		setting.dependOnIp = true
	}}
}

func Limit(every time.Duration) Option {
	return Option{fn: func(setting *rateLimitingSetting) {
		setting.rate = rate.Every(every)
	}}
}

func NewRateLimiting(path, method string, every time.Duration, opts ...Option) *rateLimitingSetting {
	opts = append(opts, Path(path), Method(method), Limit(every))
	setting := new(rateLimitingSetting)
	for _, opt := range opts {
		opt.fn(setting)
	}
	return setting
}

type Adapter interface {
	Set(string, *rate.Limiter)
	Get(string) *rate.Limiter
}

type memoryAdapter struct {
	container map[string]*rate.Limiter
	sync.Mutex
}

func (ma *memoryAdapter) Set(fp string, lim *rate.Limiter) {
	ma.Lock()
	defer ma.Unlock()
	ma.container[fp] = lim
}

func (ma *memoryAdapter) Get(fp string) *rate.Limiter {
	ma.Lock()
	defer ma.Unlock()
	lim, _ := ma.container[fp]
	return lim
}

func NewMemoryAdapter() *memoryAdapter {
	ma := new(memoryAdapter)
	ma.container = make(map[string]*rate.Limiter)
	return ma
}

func Limiter(adp Adapter, fp FingerPrint) *rate.Limiter {
	lim := adp.Get(fp.Hash())
	if lim == nil {
		lim = rate.NewLimiter(fp.Rate(), 1)
		adp.Set(fp.Hash(), lim)
	}
	return lim
}
