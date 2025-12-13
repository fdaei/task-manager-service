package http

import (
	netpprof "net/http/pprof"

	"github.com/gin-gonic/gin"
)

func NewRouter(handler *TaskHandler, enablePprof bool) *gin.Engine {
	router := gin.New()
	router.Use(gin.Logger(), gin.Recovery())

	if handler.metrics != nil {
		router.Use(handler.metrics.Middleware())
		router.GET("/metrics", handler.metrics.Handler())
	}

	if enablePprof {
		registerPprof(router)
	}

	router.POST("/users", handler.CreateUser)
	router.GET("/users/:id", handler.GetUser)

	router.POST("/tasks", handler.CreateTask)
	router.GET("/tasks", handler.ListTasks)
	router.GET("/tasks/:id", handler.GetTask)
	router.PUT("/tasks/:id", handler.UpdateTask)
	router.DELETE("/tasks/:id", handler.DeleteTask)
	RegisterOpenAPI(router)

	return router
}

func registerPprof(router *gin.Engine) {
	pprofGroup := router.Group("/debug/pprof")
	pprofGroup.GET("/", gin.WrapF(netpprof.Index))
	pprofGroup.GET("/cmdline", gin.WrapF(netpprof.Cmdline))
	pprofGroup.GET("/profile", gin.WrapF(netpprof.Profile))
	pprofGroup.POST("/symbol", gin.WrapF(netpprof.Symbol))
	pprofGroup.GET("/symbol", gin.WrapF(netpprof.Symbol))
	pprofGroup.GET("/trace", gin.WrapF(netpprof.Trace))
	pprofGroup.GET("/allocs", gin.WrapH(netpprof.Handler("allocs")))
	pprofGroup.GET("/block", gin.WrapH(netpprof.Handler("block")))
	pprofGroup.GET("/goroutine", gin.WrapH(netpprof.Handler("goroutine")))
	pprofGroup.GET("/heap", gin.WrapH(netpprof.Handler("heap")))
	pprofGroup.GET("/mutex", gin.WrapH(netpprof.Handler("mutex")))
	pprofGroup.GET("/threadcreate", gin.WrapH(netpprof.Handler("threadcreate")))
}
