package envs

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"tespkg.in/kit/log"
)

type Server struct {
	shutdown chan error

	ginEngine *gin.Engine

	args *Args
}

// replaceable set of functions for fault injection
type patchTable struct {
	configLog func(options *log.Options) error
}

func New(a *Args) (*Server, error) {
	return newServer(a, newPatchTable())
}

func newPatchTable() *patchTable {
	return &patchTable{
		configLog: log.Configure,
	}
}

func newServer(a *Args, p *patchTable) (*Server, error) {
	a.LoggingOptions.SetLogCallers("default", true)
	if err := p.configLog(a.LoggingOptions); err != nil {
		return nil, err
	}

	// Create gin engine
	ge := gin.Default()
	ge.Use(cors.New(cors.Config{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"*"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{"*"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	ge.GET("keys")
	ge.PUT("keys")
	ge.GET("key/:fully_qualified_key_name")
	ge.PUT("key/:fully_qualified_key_name")
	ge.GET("specs")
	ge.GET("spec/:name")
	ge.PUT("spec/:name")
	ge.POST("deployment")

	return &Server{
		ginEngine: ge,
		args:      a,
	}, nil
}

func (s *Server) Run() {
	s.shutdown = make(chan error)
	go func() {
		err := http.ListenAndServe(s.args.ListenAddr, s.ginEngine)

		s.shutdown <- err
	}()
}

func (s *Server) Wait() error {
	if s.shutdown == nil {
		return fmt.Errorf("server not runnig")
	}

	err := <-s.shutdown
	s.shutdown = nil
	return err
}

func (s *Server) Close() {
	log.Info("Close server")
	if s.shutdown != nil {
		_ = s.Wait()
	}

	_ = log.Sync()
}
