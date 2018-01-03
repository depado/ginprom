# ginprom
Gin Prometheus metrics exporter inspired by [github.com/zsais/go-gin-prometheus](https://github.com/zsais/go-gin-prometheus)

![Go Version](https://img.shields.io/badge/go-1.8-brightgreen.svg)
![Go Version](https://img.shields.io/badge/go-1.9-brightgreen.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/Depado/ginprom)](https://goreportcard.com/report/github.com/Depado/ginprom)
[![Build Status](https://drone.depado.eu/api/badges/Depado/ginprom/status.svg)](https://drone.depado.eu/Depado/ginprom)
[![codecov](https://codecov.io/gh/Depado/ginprom/branch/master/graph/badge.svg)](https://codecov.io/gh/Depado/ginprom)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/Depado/bfchroma/blob/master/LICENSE)


## Install

Simply run :
`go get -u github.com/Depado/ginprom`

## Differences with go-gin-prometheus

- No support for Prometheus' Push Gateway
- Options on constructor
- Adds a `path` label to get the matched route
- Ability to ignore routes

## Usage

```go
package main

import (
	"github.com/Depado/ginprom"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	p := ginprom.New(
		ginprom.Subsystem("gin"), 
		ginprom.Path("/metrics"), 
	)
	r.Use(p.Instrument())

	r.GET("/hello/:id", func(c *gin.Context) {})
	r.GET("/world/:id", func(c *gin.Context) {})
	r.Run("127.0.0.1:8080")
}
```

## Options

`Path(path string)`  
Specify the path on which the metrics are accessed  
Default : "/metrics"

`Subsystem(sub string)`  
Specify the subsystem  
Default : "gin"

`Engine(e *gin.Engine)`  
Specify the Gin engine directly when initializing. 
Saves a call to `Use(e *gin.Engine)`  
Default : `nil`

`Ignore(paths ...string)`   
Specify which paths should not be taken into account by the middleware.