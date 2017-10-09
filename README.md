# ginprom
Gin Prometheus metrics exporter inspired by [github.com/zsais/go-gin-prometheus](https://github.com/zsais/go-gin-prometheus)

![Go Version](https://img.shields.io/badge/go-1.8-brightgreen.svg)
![Go Version](https://img.shields.io/badge/go-1.9-brightgreen.svg)
[![Go Report Card](https://goreportcard.com/badge/github.com/Depado/ginprom)](https://goreportcard.com/report/github.com/Depado/ginprom)
[![Build Status](https://drone.depado.eu/api/badges/Depado/ginprom/status.svg)](https://drone.depado.eu/Depado/ginprom)
[![License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/Depado/bfchroma/blob/master/LICENSE)


## Install

Simply run :
`go get -u github.com/Depado/ginprom`

## Differences with go-gin-prometheus

- No support for Prometheus' Push Gateway
- Need to call middleware on each route
- Options on constructor
- Adds a `path` label to get the matched route

## Usage

```go
package main

import (
	"github.com/Depado/ginprom"
	"github.com/gin-gonic/gin"
)

func main() {
	r := gin.Default()
	p := ginprom.New(ginprom.Subsystem("gin"), ginprom.Path("/metrics"), ginprom.Engine(r))

	r.GET("/hello/:id", p.Instrument("/hello/:id"), func(c *gin.Context) {})
	r.GET("/world/:id", p.Instrument("/world/:id"), func(c *gin.Context) {})
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
Specify the Gin engine directly when intializing. 
Saves a call to `Use(e *gin.Engine)`  
Default : `nil`