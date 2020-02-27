package envs

import (
	"github.com/gin-gonic/gin"
	"tespkg.in/envs/pkg/spec"
	"tespkg.in/envs/pkg/store"
)

type Handler struct {
	store.Store
	spec.Handler
}

func NewHandler(s store.Store) *Handler {
	return &Handler{
		Store:   s,
		Handler: spec.NewHandler(s),
	}
}

func (h *Handler) GetKeys(c *gin.Context) {}

func (h *Handler) PutKeys(c *gin.Context) {}

func (h *Handler) GetKey(c *gin.Context) {}

func (h *Handler) PutKey(c *gin.Context) {}

func (h *Handler) GetSpecs(c *gin.Context) {}

func (h *Handler) GetSpec(c *gin.Context) {}

func (h *Handler) PutSpec(c *gin.Context) {}

func (h *Handler) PatchSpec(c *gin.Context) {}

func (h *Handler) PostDeployment(c *gin.Context) {}
