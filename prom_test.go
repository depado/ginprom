package ginprom

import (
	"fmt"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/appleboy/gofight/v2"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	io_prometheus_client "github.com/prometheus/client_model/go"
	"github.com/stretchr/testify/assert"
)

func unregister(p *Prometheus) {
	prometheus.Unregister(p.reqCnt)
	prometheus.Unregister(p.reqDur)
	prometheus.Unregister(p.reqSz)
	prometheus.Unregister(p.resSz)
}

func init() {
	gin.SetMode(gin.TestMode)
}

func TestPrometheus_Use(t *testing.T) {
	p := New()
	r := gin.New()

	p.Use(r)

	assert.Equal(t, 1, len(r.Routes()), "only one route should be added")
	assert.NotNil(t, p.Engine, "the engine should not be empty")
	assert.Equal(t, r, p.Engine, "used router should be the same")
	assert.Equal(t, r.Routes()[0].Path, p.MetricsPath, "the path should match the metrics path")
	unregister(p)
}

// Set the path (endpoint) where the metrics will be served
func ExamplePath() {
	r := gin.New()
	p := New(Engine(r), Path("/metrics"))
	r.Use(p.Instrument())
}

func TestPath(t *testing.T) {
	p := New()
	assert.Equal(t, p.MetricsPath, defaultPath, "no usage of path should yield default path")
	unregister(p)

	valid := []string{"/metrics", "/home", "/x/x", ""}
	for _, tt := range valid {
		p = New(Path(tt))
		assert.Equal(t, p.MetricsPath, tt)
		unregister(p)
	}
}

// Set a secret token that is required to access the endpoint
func ExampleToken() {
	r := gin.New()
	p := New(Engine(r), Token("supersecrettoken"))
	r.Use(p.Instrument())
}

func TestToken(t *testing.T) {
	valid := []string{"token1", "token2", ""}
	for _, tt := range valid {
		p := New(Token(tt))
		assert.Equal(t, tt, p.Token)
		unregister(p)
	}
}

func TestEngine(t *testing.T) {
	r := gin.New()
	p := New(Engine(r))
	assert.Equal(t, 1, len(r.Routes()), "only one route should be added")
	assert.NotNil(t, p.Engine, "engine should not be nil")
	assert.Equal(t, r.Routes()[0].Path, p.MetricsPath, "the path should match the metrics path")
	assert.Equal(t, p.MetricsPath, defaultPath, "path should be default")
	unregister(p)
}

func TestRegistry(t *testing.T) {
	registry := prometheus.NewRegistry()

	p := New(Registry(registry))
	assert.Equal(t, p.Registry, registry)
}

func TestHandlerNameFunc(t *testing.T) {
	r := gin.New()
	registry := prometheus.NewRegistry()
	handler := "handler_label_should_have_this_value"
	lhandler := fmt.Sprintf("handler=%q", handler)

	p := New(
		HandlerNameFunc(func(c *gin.Context) string {
			return handler
		}),
		Registry(registry),
		Engine(r),
	)

	r.Use(p.Instrument())

	r.GET("/", func(context *gin.Context) {
		context.Status(http.StatusOK)
	})

	g := gofight.New()

	g.GET("/").Run(r, func(response gofight.HTTPResponse, request gofight.HTTPRequest) {
		assert.Equal(t, response.Code, http.StatusOK)
	})

	g.GET(p.MetricsPath).Run(r, func(response gofight.HTTPResponse, request gofight.HTTPRequest) {
		assert.Equal(t, response.Code, http.StatusOK)
		assert.Contains(t, response.Body.String(), lhandler)
	})
}

func TestHandlerOpts(t *testing.T) {
	r := gin.New()
	registry := prometheus.NewRegistry()

	p := New(
		HandlerOpts(promhttp.HandlerOpts{Timeout: time.Nanosecond}),
		Registry(registry),
		Engine(r),
	)

	r.Use(p.Instrument())

	r.GET("/", func(context *gin.Context) {
		context.Status(http.StatusServiceUnavailable)
	})

	g := gofight.New()

	g.GET("/").Run(r, func(response gofight.HTTPResponse, request gofight.HTTPRequest) {
		assert.Equal(t, response.Code, http.StatusServiceUnavailable)
	})

	g.GET(p.MetricsPath).Run(r, func(response gofight.HTTPResponse, request gofight.HTTPRequest) {
		assert.Equal(t, response.Code, http.StatusServiceUnavailable)
	})
}

func TestRequestPathFunc(t *testing.T) {
	r := gin.New()
	registry := prometheus.NewRegistry()

	correctPath := fmt.Sprintf("path=%q", "/some/path")
	unknownPath := fmt.Sprintf("path=%q", "<unknown>")

	p := New(
		RequestPathFunc(func(c *gin.Context) string {
			if fullpath := c.FullPath(); fullpath != "" {
				return fullpath
			}
			return "<unknown>"
		}),
		Engine(r),
		Registry(registry),
	)

	r.Use(p.Instrument())

	r.GET("/some/path", func(context *gin.Context) {
		context.Status(http.StatusOK)
	})

	g := gofight.New()
	g.GET("/some/path").Run(r, func(response gofight.HTTPResponse, request gofight.HTTPRequest) {
		assert.Equal(t, response.Code, http.StatusOK)
	})
	g.GET("/some/other/path").Run(r, func(response gofight.HTTPResponse, request gofight.HTTPRequest) {
		assert.Equal(t, response.Code, http.StatusNotFound)
	})

	g.GET(p.MetricsPath).Run(r, func(response gofight.HTTPResponse, request gofight.HTTPRequest) {
		assert.Equal(t, response.Code, http.StatusOK)
		assert.Contains(t, response.Body.String(), correctPath)
		assert.Contains(t, response.Body.String(), unknownPath)
	})
}

func TestNamespace(t *testing.T) {
	p := New()
	assert.Equal(t, p.Namespace, defaultNs, "namespace should be default")
	unregister(p)

	tests := []string{
		"test",
		"",
		"_",
	}
	for _, test := range tests {
		p = New(Namespace(test))
		assert.Equal(t, p.Namespace, test, "should match")
		unregister(p)
	}
}

func TestRequestCounterMetricName(t *testing.T) {
	p := New()
	assert.Equal(t, p.RequestCounterMetricName, defaultReqCntMetricName, "subsystem should be default")
	unregister(p)

	p = New(RequestCounterMetricName("another_req_cnt_metric_name"))
	assert.Equal(t, p.RequestCounterMetricName, "another_req_cnt_metric_name", "should match")
	unregister(p)
}

func TestRequestDurationMetricName(t *testing.T) {
	p := New()
	assert.Equal(t, p.RequestDurationMetricName, defaultReqDurMetricName, "subsystem should be default")
	unregister(p)

	p = New(RequestDurationMetricName("another_req_dur_metric_name"))
	assert.Equal(t, p.RequestDurationMetricName, "another_req_dur_metric_name", "should match")
	unregister(p)
}

func TestRequestSizeMetricName(t *testing.T) {
	p := New()
	assert.Equal(t, p.RequestSizeMetricName, defaultReqSzMetricName, "subsystem should be default")
	unregister(p)

	p = New(RequestSizeMetricName("another_req_sz_metric_name"))
	assert.Equal(t, p.RequestSizeMetricName, "another_req_sz_metric_name", "should match")
	unregister(p)
}

func TestResponseSizeMetricName(t *testing.T) {
	p := New()
	assert.Equal(t, p.ResponseSizeMetricName, defaultResSzMetricName, "subsystem should be default")
	unregister(p)

	p = New(ResponseSizeMetricName("another_res_sz_metric_name"))
	assert.Equal(t, p.ResponseSizeMetricName, "another_res_sz_metric_name", "should match")
	unregister(p)
}

func TestSubsystem(t *testing.T) {
	p := New()
	assert.Equal(t, p.Subsystem, defaultSys, "subsystem should be default")
	unregister(p)

	tests := []string{
		"test",
		"",
		"_",
	}
	for _, test := range tests {
		p = New(Subsystem(test))
		assert.Equal(t, p.Subsystem, test, "should match")
		unregister(p)
	}
}

func TestUse(t *testing.T) {
	r := gin.New()
	p := New()

	g := gofight.New()
	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusNotFound, r.Code)
	})

	p.Use(r)
	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
	})
	unregister(p)
}

func TestBucketSize(t *testing.T) {
	p := New()
	assert.Nil(t, p.BucketsSize, "namespace should be default")
	unregister(p)

	bs := []float64{.005, .01, .025, .05, .1, .25, .5, 1, 2.5, 5, 10}
	p = New(BucketSize(bs))
	assert.Equal(t, p.BucketsSize, bs, "should match")
	unregister(p)
}

func TestInstrument(t *testing.T) {
	r := gin.New()
	p := New(Engine(r))
	r.Use(p.Instrument())
	path := "/user/:id"
	lpath := fmt.Sprintf(`path="%s"`, path)

	r.GET(path, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
	})

	g := gofight.New()
	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
		assert.NotContains(t, r.Body.String(), prometheus.BuildFQName(p.Namespace, p.Subsystem, "requests_total"))
		assert.NotContains(t, r.Body.String(), lpath, "path must not be present in the response")
	})

	g.GET("/user/10").Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) { assert.Equal(t, http.StatusOK, r.Code) })

	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
		assert.Contains(t, r.Body.String(), prometheus.BuildFQName(p.Namespace, p.Subsystem, "requests_total"))
		assert.Contains(t, r.Body.String(), lpath, "path must be present in the response")
		assert.NotContains(t, r.Body.String(), `path="/user/10"`, "raw path must not be present")
	})

	unregister(p)
}

func TestThreadedInstrument(t *testing.T) {
	r := gin.New()
	p := New(Engine(r))
	r.Use(p.Instrument())
	path := "/user/:id"
	lpath := fmt.Sprintf(`path="%s"`, path)

	r.GET(path, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"id": c.Param("id")})
	})

	var wg sync.WaitGroup
	for n := 0; n < 10; n++ {
		go func(wg *sync.WaitGroup) {
			g := gofight.New()

			g.GET("/user/10").Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) { assert.Equal(t, http.StatusOK, r.Code) })

			g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
				assert.Equal(t, http.StatusOK, r.Code)
				assert.Contains(t, r.Body.String(), prometheus.BuildFQName(p.Namespace, p.Subsystem, "requests_total"))
				assert.Contains(t, r.Body.String(), lpath, "path must be present in the response")
				assert.NotContains(t, r.Body.String(), `path="/user/10"`, "raw path must not be present")
			})
			wg.Done()
		}(&wg)
		wg.Add(1)
	}
	wg.Wait()
	unregister(p)
}

func TestEmptyRouter(t *testing.T) {
	r := gin.New()
	p := New()

	r.Use(p.Instrument())
	r.GET("/", func(c *gin.Context) { c.JSON(http.StatusOK, gin.H{}) })

	g := gofight.New()
	assert.NotPanics(t, func() {
		g.GET("/").Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {})
	})
	unregister(p)
}

func TestCustomCounterMetrics(t *testing.T) {
	r := gin.New()
	p := New(Engine(r), Registry(prometheus.NewRegistry()), CustomCounterLabels([]string{"client_id", "tenant_id"}, func(c *gin.Context) map[string]string {
		clientId := c.GetHeader("X-Client-ID")
		if clientId == "" {
			clientId = "unknown"
		}
		tenantId := c.GetHeader("X-Tenant-ID")
		if tenantId == "" {
			tenantId = "unknown"
		}
		return map[string]string{
			"client_id": clientId,
			"tenant_id": tenantId,
		}
	}))
	r.Use(p.Instrument())

	r.GET("/ping", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	g := gofight.New()
	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
		assert.NotContains(t, r.Body.String(), prometheus.BuildFQName(p.Namespace, p.Subsystem, "requests_total"))
		assert.NotContains(t, r.Body.String(), "client_id")
		assert.NotContains(t, r.Body.String(), "tenant_id")
	})

	g.GET("/ping").Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) { assert.Equal(t, http.StatusOK, r.Code) })

	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		body := r.Body.String()
		assert.Equal(t, http.StatusOK, r.Code)
		assert.Contains(t, body, prometheus.BuildFQName(p.Namespace, p.Subsystem, "requests_total"))
		assert.Contains(t, r.Body.String(), "client_id=\"unknown\"")
		assert.Contains(t, r.Body.String(), "tenant_id=\"unknown\"")
		assert.NotContains(t, r.Body.String(), "client_id=\"client-id\"")
		assert.NotContains(t, r.Body.String(), "tenant_id=\"tenant-id\"")
	})

	g.GET("/ping").
		SetHeader(gofight.H{"X-Client-Id": "client-id", "X-Tenant-Id": "tenant-id"}).
		Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) { assert.Equal(t, http.StatusOK, r.Code) })

	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		body := r.Body.String()
		assert.Equal(t, http.StatusOK, r.Code)
		assert.Contains(t, body, prometheus.BuildFQName(p.Namespace, p.Subsystem, "requests_total"))
		assert.Contains(t, r.Body.String(), "client_id=\"unknown\"")
		assert.Contains(t, r.Body.String(), "tenant_id=\"unknown\"")
		assert.Contains(t, r.Body.String(), "client_id=\"client-id\"")
		assert.Contains(t, r.Body.String(), "tenant_id=\"tenant-id\"")
	})
	unregister(p)
}

func TestCustomHistogram(t *testing.T) {
	r := gin.New()
	p := New(Engine(r), Registry(prometheus.NewRegistry()))
	p.AddCustomHistogram("request_latency", "test histogram", []string{"url", "method"})
	r.Use(p.Instrument())
	defer unregister(p)

	r.GET("/ping", func(c *gin.Context) {
		err := p.AddCustomHistogramValue("request_latency", []string{"http://example.com/status", "GET"}, 0.45)
		assert.NoError(t, err)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/pong", func(c *gin.Context) {
		err := p.AddCustomHistogramValue("request_latency", []string{"http://example.com/status", "GET"}, 9.56)
		assert.NoError(t, err)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})
	r.GET("/error", func(c *gin.Context) {
		// Metric not found
		err := p.AddCustomHistogramValue("invalid", []string{}, 9.56)
		assert.Error(t, err)
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	expectedLines := []string{
		`gin_gonic_request_latency_bucket{method="GET",url="http://example.com/status",le="0.005"} 0`,
		`gin_gonic_request_latency_bucket{method="GET",url="http://example.com/status",le="0.01"} 0`,
		`gin_gonic_request_latency_bucket{method="GET",url="http://example.com/status",le="0.025"} 0`,
		`gin_gonic_request_latency_bucket{method="GET",url="http://example.com/status",le="0.05"} 0`,
		`gin_gonic_request_latency_bucket{method="GET",url="http://example.com/status",le="0.1"} 0`,
		`gin_gonic_request_latency_bucket{method="GET",url="http://example.com/status",le="0.25"} 0`,
		`gin_gonic_request_latency_bucket{method="GET",url="http://example.com/status",le="0.5"} 1`,
		`gin_gonic_request_latency_bucket{method="GET",url="http://example.com/status",le="1"} 1`,
		`gin_gonic_request_latency_bucket{method="GET",url="http://example.com/status",le="2.5"} 1`,
		`gin_gonic_request_latency_bucket{method="GET",url="http://example.com/status",le="5"} 1`,
		`gin_gonic_request_latency_bucket{method="GET",url="http://example.com/status",le="10"} 2`,
		`gin_gonic_request_latency_bucket{method="GET",url="http://example.com/status",le="+Inf"} 2`,
		`gin_gonic_request_latency_sum{method="GET",url="http://example.com/status"} 10.01`,
		`gin_gonic_request_latency_count{method="GET",url="http://example.com/status"} 2`,
	}

	g := gofight.New()
	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
		assert.NotContains(t, r.Body.String(), prometheus.BuildFQName(p.Namespace, p.Subsystem, "request_latency"))

		for _, line := range expectedLines {
			assert.NotContains(t, r.Body.String(), line)
		}
	})

	g.GET("/ping").Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
	})
	g.GET("/pong").Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
	})
	g.GET("/error").Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
	})

	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
		assert.Contains(t, r.Body.String(), prometheus.BuildFQName(p.Namespace, p.Subsystem, "request_latency"))

		for _, line := range expectedLines {
			assert.Contains(t, r.Body.String(), line)
		}
	})
}

func TestCustomNativeHistogram(t *testing.T) {
	r := gin.New()
	registry := prometheus.NewRegistry()
	p := New(Engine(r), Registry(registry), NativeHistogram(true))
	p.AddCustomHistogram("custom_histogram", "test histogram", []string{"url", "method"})
	r.Use(p.Instrument())
	defer unregister(p)

	err := p.AddCustomHistogramValue("custom_histogram", []string{"http://example.com/status", "GET"}, 0.45)
	assert.Nil(t, err)

	mfs, err := registry.Gather()
	assert.Nil(t, err)

	found := false

	for _, mf := range mfs {
		if mf.GetType() == io_prometheus_client.MetricType_HISTOGRAM {
			for _, m := range mf.Metric {
				if mf.GetName() == "gin_gonic_custom_histogram" {
					found = true
					assert.Equal(t, int32(3), m.GetHistogram().GetSchema())
					assert.Equal(t, uint64(0x1), m.GetHistogram().GetSampleCount())
					assert.Equal(t, 0.45, m.GetHistogram().GetSampleSum())
				}
			}
		}
	}

	assert.True(t, found)
}

func TestIgnore(t *testing.T) {
	r := gin.New()
	ipath := "/ping"
	lipath := fmt.Sprintf(`path="%s"`, ipath)
	p := New(Engine(r), Ignore(ipath))
	r.Use(p.Instrument())

	r.GET(ipath, func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	g := gofight.New()
	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
		assert.NotContains(t, r.Body.String(), prometheus.BuildFQName(p.Namespace, p.Subsystem, "requests_total"))
	})

	g.GET("/ping").Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) { assert.Equal(t, http.StatusOK, r.Code) })

	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
		assert.NotContains(t, r.Body.String(), prometheus.BuildFQName(p.Namespace, p.Subsystem, "requests_total"))
		assert.NotContains(t, r.Body.String(), lipath, "ignored path must not be present")
	})
	unregister(p)
}

func TestMetricsPathIgnored(t *testing.T) {
	r := gin.New()
	p := New(Engine(r))
	r.Use(p.Instrument())

	g := gofight.New()
	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
		assert.NotContains(t, r.Body.String(), prometheus.BuildFQName(p.Namespace, p.Subsystem, "requests_total"))
	})
	unregister(p)
}

func TestMetricsBearerToken(t *testing.T) {
	r := gin.New()
	p := New(Engine(r), Token("test-1234"))
	r.Use(p.Instrument())

	g := gofight.New()

	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusUnauthorized, r.Code)
		assert.Equal(t, ErrInvalidToken.Error(), r.Body.String())
	})

	g.GET(p.MetricsPath).
		SetHeader(gofight.H{
			"Authorization": "Bearer " + "test-1234-5678",
		}).
		Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusUnauthorized, r.Code)
			assert.Equal(t, ErrInvalidToken.Error(), r.Body.String())
		})

	g.GET(p.MetricsPath).
		SetHeader(gofight.H{
			"Authorization": "Bearer " + "test-1234",
		}).
		Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusOK, r.Code)
			assert.NotContains(t, r.Body.String(), prometheus.BuildFQName(p.Namespace, p.Subsystem, "requests_total"))
		})
	unregister(p)
}

func TestCustomCounterErr(t *testing.T) {
	p := New()
	assert.Equal(t, p.IncrementCounterValue("not_found", []string{"some", "labels"}), ErrCustomCounter)
	assert.Equal(t, p.AddCounterValue("not_found", []string{"some", "labels"}, 1.), ErrCustomCounter)
	unregister(p)
}

func TestCustomGaugeErr(t *testing.T) {
	p := New()
	assert.Equal(t, p.IncrementGaugeValue("not_found", []string{"some", "labels"}), ErrCustomGauge)
	assert.Equal(t, p.DecrementGaugeValue("not_found", []string{"some", "labels"}), ErrCustomGauge)
	assert.Equal(t, p.AddGaugeValue("not_found", []string{"some", "labels"}, 1.), ErrCustomGauge)
	assert.Equal(t, p.SubGaugeValue("not_found", []string{"some", "labels"}, 1.), ErrCustomGauge)
	assert.Equal(t, p.SetGaugeValue("not_found", []string{"some", "labels"}, 1.), ErrCustomGauge)
	unregister(p)
}

func TestInstrumentCustomCounter(t *testing.T) {
	var helpText = "help text"
	var labels = []string{"label1"}
	var name = "custom_counter"

	r := gin.New()
	p := New(Engine(r))
	p.AddCustomCounter(name, helpText, labels)
	r.Use(p.Instrument())

	r.GET("/inc", func(c *gin.Context) {
		err := p.IncrementCounterValue(name, labels)
		assert.NoError(t, err, "should not fail with same Counter name")
		c.Status(http.StatusOK)
	})

	r.GET("/add", func(c *gin.Context) {
		err := p.AddCounterValue(name, labels, 10)
		assert.NoError(t, err, "should not fail with same Counter name")
		c.Status(http.StatusOK)
	})

	g := gofight.New()

	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.NotContains(t, r.Body.String(), fmt.Sprintf(`# HELP gin_gonic_%s %s`, name, helpText))
		assert.NotContains(t, r.Body.String(), fmt.Sprintf(`gin_gonic_%s{%s="%s"} 0`, name, labels[0], labels[0]))
		assert.Equal(t, http.StatusOK, r.Code)
	})

	g.GET("/inc").Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
	})

	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Contains(t, r.Body.String(), fmt.Sprintf(`# HELP gin_gonic_%s %s`, name, helpText))
		assert.Contains(t, r.Body.String(), fmt.Sprintf(`gin_gonic_%s{%s="%s"} 1`, name, labels[0], labels[0]))
		assert.Equal(t, http.StatusOK, r.Code)
	})

	g.GET("/add").Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
	})

	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Contains(t, r.Body.String(), fmt.Sprintf(`# HELP gin_gonic_%s %s`, name, helpText))
		assert.Contains(t, r.Body.String(), fmt.Sprintf(`gin_gonic_%s{%s="%s"} 11`, name, labels[0], labels[0]))
		assert.Equal(t, http.StatusOK, r.Code)
	})

	unregister(p)
}

func TestInstrumentCustomGauge(t *testing.T) {
	var helpText = "help text"
	var labels = []string{"label1"}
	var name = "custom_gauge"

	r := gin.New()
	p := New(Engine(r))
	p.AddCustomGauge(name, helpText, labels)
	r.Use(p.Instrument())

	r.GET("/inc", func(c *gin.Context) {
		err := p.IncrementGaugeValue(name, labels)
		assert.NoError(t, err, "should not fail with same gauge name")
		c.Status(http.StatusOK)
	})

	r.GET("/dec", func(c *gin.Context) {
		err := p.DecrementGaugeValue(name, labels)
		assert.NoError(t, err, "should not fail with same gauge name")
		c.Status(http.StatusOK)
	})

	r.GET("/set", func(c *gin.Context) {
		err := p.SetGaugeValue(name, labels, 10)
		assert.NoError(t, err, "should not fail with same gauge name")
		c.Status(http.StatusOK)
	})

	r.GET("/add", func(c *gin.Context) {
		err := p.AddGaugeValue(name, labels, 10)
		assert.NoError(t, err, "should not fail with same gauge name")
		c.Status(http.StatusOK)
	})

	r.GET("/sub", func(c *gin.Context) {
		err := p.SubGaugeValue(name, labels, 10)
		assert.NoError(t, err, "should not fail with same gauge name")
		c.Status(http.StatusOK)
	})

	g := gofight.New()

	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.NotContains(t, r.Body.String(), fmt.Sprintf(`# HELP gin_gonic_%s %s`, name, helpText))
		assert.NotContains(t, r.Body.String(), fmt.Sprintf(`gin_gonic_%s{%s="%s"} 0`, name, labels[0], labels[0]))
		assert.Equal(t, http.StatusOK, r.Code)
	})

	g.GET("/inc").Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
	})

	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Contains(t, r.Body.String(), fmt.Sprintf(`# HELP gin_gonic_%s %s`, name, helpText))
		assert.Contains(t, r.Body.String(), fmt.Sprintf(`gin_gonic_%s{%s="%s"} 1`, name, labels[0], labels[0]))
		assert.Equal(t, http.StatusOK, r.Code)
	})

	g.GET("/dec").Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
	})

	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Contains(t, r.Body.String(), fmt.Sprintf(`# HELP gin_gonic_%s %s`, name, helpText))
		assert.Contains(t, r.Body.String(), fmt.Sprintf(`gin_gonic_%s{%s="%s"} 0`, name, labels[0], labels[0]))
		assert.Equal(t, http.StatusOK, r.Code)
	})

	g.GET("/set").Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
	})

	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Contains(t, r.Body.String(), fmt.Sprintf(`# HELP gin_gonic_%s %s`, name, helpText))
		assert.Contains(t, r.Body.String(), fmt.Sprintf(`gin_gonic_%s{%s="%s"} 10`, name, labels[0], labels[0]))
		assert.Equal(t, http.StatusOK, r.Code)
	})

	g.GET("/add").Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
	})

	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Contains(t, r.Body.String(), fmt.Sprintf(`# HELP gin_gonic_%s %s`, name, helpText))
		assert.Contains(t, r.Body.String(), fmt.Sprintf(`gin_gonic_%s{%s="%s"} 20`, name, labels[0], labels[0]))
		assert.Equal(t, http.StatusOK, r.Code)
	})

	g.GET("/sub").Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
	})

	g.GET(p.MetricsPath).Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Contains(t, r.Body.String(), fmt.Sprintf(`# HELP gin_gonic_%s %s`, name, helpText))
		assert.Contains(t, r.Body.String(), fmt.Sprintf(`gin_gonic_%s{%s="%s"} 10`, name, labels[0], labels[0]))
		assert.Equal(t, http.StatusOK, r.Code)
	})

	unregister(p)
}

func TestInstrumentCustomMetricsErrors(t *testing.T) {
	r := gin.New()
	p := New(Engine(r))
	r.Use(p.Instrument())

	r.GET("/err", func(c *gin.Context) {
		err := p.IncrementGaugeValue("notfound", []string{})
		assert.EqualError(t, err, "error finding custom gauge")
		c.Status(http.StatusOK)
	})
	g := gofight.New()

	g.GET("/err").Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
		assert.Equal(t, http.StatusOK, r.Code)
	})

	unregister(p)
}

func TestMultipleGinWithDifferentRegistry(t *testing.T) {
	// with different registries we don't panic because of multiple metric registration attempt
	r1 := gin.New()
	p1 := New(Engine(r1), Registry(prometheus.NewRegistry()))
	r1.Use(p1.Instrument())

	r2 := gin.New()
	p2 := New(Engine(r2), Registry(prometheus.NewRegistry()))
	r2.Use(p2.Instrument())
}

func TestCustomGaugeCorrectRegistry(t *testing.T) {
	reg := prometheus.NewRegistry()
	p := New(Registry(reg))

	p.AddCustomGauge("some_gauge", "", nil)
	// increment the gauge value so it is reported by Gather
	err := p.IncrementGaugeValue("some_gauge", nil)
	assert.Nil(t, err)

	fams, err := reg.Gather()
	assert.Nil(t, err)
	assert.Len(t, fams, 3)

	assert.Condition(t, func() (success bool) {
		for _, fam := range fams {
			if fam.GetName() == fmt.Sprintf("%s_%s_some_gauge", p.Namespace, p.Subsystem) {
				return true
			}
		}
		return false
	})
}

func TestCustomCounterCorrectRegistry(t *testing.T) {
	reg := prometheus.NewRegistry()
	p := New(Registry(reg))

	p.AddCustomCounter("some_counter", "", nil)
	// increment the counter value so it is reported by Gather
	err := p.IncrementCounterValue("some_counter", nil)
	assert.Nil(t, err)

	fams, err := reg.Gather()
	assert.Nil(t, err)
	assert.Len(t, fams, 3)

	assert.Condition(t, func() (success bool) {
		for _, fam := range fams {
			if fam.GetName() == fmt.Sprintf("%s_%s_some_counter", p.Namespace, p.Subsystem) {
				return true
			}
		}
		return false
	})
}
