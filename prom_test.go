package ginprom

import (
	"fmt"
	"net/http"
	"sync"
	"testing"

	"github.com/appleboy/gofight/v2"
	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus"
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
		assert.Equal(t, errInvalidToken.Error(), r.Body.String())
	})

	g.GET(p.MetricsPath).
		SetHeader(gofight.H{
			"Authorization": "Bearer " + "test-1234-5678",
		}).
		Run(r, func(r gofight.HTTPResponse, rq gofight.HTTPRequest) {
			assert.Equal(t, http.StatusUnauthorized, r.Code)
			assert.Equal(t, errInvalidToken.Error(), r.Body.String())
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
