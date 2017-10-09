# ginprom
Gin Prometheus metrics exporter inspired by [github.com/zsais/go-gin-prometheus](https://github.com/zsais/go-gin-prometheus)

[![Build Status](https://drone.depado.eu/api/badges/Depado/ginprom/status.svg)](https://drone.depado.eu/Depado/ginprom)


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