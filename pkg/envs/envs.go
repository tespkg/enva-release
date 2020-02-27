package envs

import (
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"tespkg.in/envs/pkg/store"
	"tespkg.in/envs/pkg/store/consul"
	"tespkg.in/envs/pkg/store/etcd"
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

	// Connect to env store, i.e, consul.
	u, err := url.Parse(a.Dsn)
	if err != nil {
		return nil, fmt.Errorf("invalid env store dsn: %v", a.Dsn)
	}
	var s store.Store
	switch strings.ToLower(u.Scheme) {
	case "http", "https":
		s, err = consul.NewStore(a.Dsn)
		if err != nil {
			return nil, fmt.Errorf("initiate consul env store failed: :%v", err)
		}
	case "etcd":
		s, err = etcd.NewStore(a.Dsn)
		if err != nil {
			return nil, fmt.Errorf("initiate etcd env store failed: %v", err)
		}
	default:
		return nil, fmt.Errorf("unknown env store schema: %v", u.Scheme)
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

	// Create APIs handler
	handler := NewHandler(s)
	ge.GET("keys", handler.GetKeys)
	ge.PUT("keys", handler.PutKeys)
	ge.PUT("key", handler.PutKey)
	ge.GET("key/*fully_qualified_key_name", handler.GetKey)
	ge.GET("specs", handler.GetSpecs)
	ge.PUT("spec", handler.PutSpec)
	ge.GET("spec/:name", handler.GetSpec)
	ge.PATCH("spec/:name", handler.PatchSpec)
	ge.POST("deployment", handler.PostDeployment)

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
