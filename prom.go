package ginprom

import (
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var defaultPath = "/metrics"
var defaultNs = "gin"
var defaultSys = "gonic"
var errInvalidToken = errors.New("Invalid or missing token")

type pmap struct {
	sync.RWMutex
	values map[string]string
}

type pmapb struct {
	sync.RWMutex
	values map[string]bool
}

// Prometheus contains the metrics gathered by the instance and its path
type Prometheus struct {
	reqCnt               *prometheus.CounterVec
	reqDur, reqSz, resSz prometheus.Summary

	MetricsPath string
	Namespace   string
	Subsystem   string
	Token       string
	Ignored     pmapb
	Engine      *gin.Engine
	PathMap     pmap
}

// Path is an option allowing to set the metrics path when intializing with New.
// Example : ginprom.New(ginprom.Path("/mymetrics"))
func Path(path string) func(*Prometheus) {
	return func(p *Prometheus) {
		p.MetricsPath = path
	}
}

// Ignore is used to disable instrumentation on some routes
func Ignore(paths ...string) func(*Prometheus) {
	return func(p *Prometheus) {
		p.Ignored.Lock()
		defer p.Ignored.Unlock()
		for _, path := range paths {
			p.Ignored.values[path] = true
		}
	}
}

// Subsystem is an option allowing to set the subsystem when intitializing
// with New.
// Example : ginprom.New(ginprom.Subsystem("my_system"))
func Subsystem(sub string) func(*Prometheus) {
	return func(p *Prometheus) {
		p.Subsystem = sub
	}
}

// Namespace is an option allowing to set the namespace when intitializing
// with New.
// Example : ginprom.New(ginprom.Namespace("my_namespace"))
func Namespace(ns string) func(*Prometheus) {
	return func(p *Prometheus) {
		p.Namespace = ns
	}
}

// Token is an option allowing to set the bearer token in prometheus
// with New.
// Example : ginprom.New(ginprom.Token("your_custom_token"))
func Token(token string) func(*Prometheus) {
	return func(p *Prometheus) {
		p.Token = token
	}
}

// Engine is an option allowing to set the gin engine when intializing with New.
// Example :
// r := gin.Default()
// p := ginprom.New(Engine(r))
func Engine(e *gin.Engine) func(*Prometheus) {
	return func(p *Prometheus) {
		p.Engine = e
	}
}

// New will initialize a new Prometheus instance with the given options.
// If no options are passed, sane defaults are used.
// If a router is passed using the Engine() option, this instance will
// automatically bind to it.
func New(options ...func(*Prometheus)) *Prometheus {
	p := &Prometheus{
		MetricsPath: defaultPath,
		Namespace:   defaultNs,
		Subsystem:   defaultSys,
	}
	p.Ignored.values = make(map[string]bool)
	for _, option := range options {
		option(p)
	}
	p.register()
	if p.Engine != nil {
		p.Engine.GET(p.MetricsPath, prometheusHandler(p.Token))
	}

	return p
}

func (p *Prometheus) update() {
	p.PathMap.Lock()
	p.Ignored.RLock()
	if p.PathMap.values == nil {
		p.PathMap.values = make(map[string]string)
	}
	defer func() {
		p.PathMap.Unlock()
		p.Ignored.RUnlock()
	}()
	if p.Engine != nil {
		for _, ri := range p.Engine.Routes() {
			if _, ok := p.Ignored.values[ri.Path]; ok {
				continue
			}
			p.PathMap.values[ri.Handler] = ri.Path
		}
	}
}

func (p *Prometheus) get(handler string) (string, bool) {
	p.PathMap.RLock()
	defer p.PathMap.RUnlock()
	in, ok := p.PathMap.values[handler]
	return in, ok
}

func (p *Prometheus) register() {
	p.reqCnt = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: p.Namespace,
			Subsystem: p.Subsystem,
			Name:      "requests_total",
			Help:      "How many HTTP requests processed, partitioned by status code and HTTP method.",
		},
		[]string{"code", "method", "handler", "host", "path"},
	)
	prometheus.MustRegister(p.reqCnt)

	p.reqDur = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Namespace: p.Namespace,
			Subsystem: p.Subsystem,
			Name:      "request_duration_seconds",
			Help:      "The HTTP request latencies in seconds.",
		},
	)
	prometheus.MustRegister(p.reqDur)

	p.reqSz = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Namespace: p.Namespace,
			Subsystem: p.Subsystem,
			Name:      "request_size_bytes",
			Help:      "The HTTP request sizes in bytes.",
		},
	)
	prometheus.MustRegister(p.reqSz)

	p.resSz = prometheus.NewSummary(
		prometheus.SummaryOpts{
			Namespace: p.Namespace,
			Subsystem: p.Subsystem,
			Name:      "response_size_bytes",
			Help:      "The HTTP response sizes in bytes.",
		},
	)
	prometheus.MustRegister(p.resSz)
}

// Instrument is a gin middleware that can be used to generate metrics for a
// single handler
func (p *Prometheus) Instrument() gin.HandlerFunc {
	return func(c *gin.Context) {
		p.PathMap.RLock()
		if p.PathMap.values == nil {
			p.PathMap.RUnlock()
			p.update()
		} else {
			p.PathMap.RUnlock()
		}
		var path string
		var found bool

		start := time.Now()

		if path, found = p.get(c.HandlerName()); !found {
			c.Next()
			return
		}
		reqSz := computeApproximateRequestSize(c.Request)

		c.Next()

		status := strconv.Itoa(c.Writer.Status())
		elapsed := float64(time.Since(start)) / float64(time.Second)
		resSz := float64(c.Writer.Size())

		p.reqDur.Observe(elapsed)
		p.reqCnt.WithLabelValues(status, c.Request.Method, c.HandlerName(), c.Request.Host, path).Inc()
		p.reqSz.Observe(float64(reqSz))
		p.resSz.Observe(resSz)
	}
}

// Use is a method that should be used if the engine is set after middleware
// initialization
func (p *Prometheus) Use(e *gin.Engine) {
	e.GET(p.MetricsPath, prometheusHandler(p.Token))
	p.Engine = e
}

func prometheusHandler(token string) gin.HandlerFunc {
	h := promhttp.Handler()
	return func(c *gin.Context) {
		if token == "" {
			h.ServeHTTP(c.Writer, c.Request)
			return
		}

		header := c.Request.Header.Get("Authorization")

		if header == "" {
			c.String(http.StatusUnauthorized, errInvalidToken.Error())
			return
		}

		bearer := fmt.Sprintf("Bearer %s", token)

		if header != bearer {
			c.String(http.StatusUnauthorized, errInvalidToken.Error())
			return
		}

		h.ServeHTTP(c.Writer, c.Request)
	}
}
