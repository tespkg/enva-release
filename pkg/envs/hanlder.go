package envs

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

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

func (h *Handler) GetKeys(c *gin.Context) {
	kind := c.Query("kind")

	var kvals store.KeyVals
	var err error
	if kind == "" {
		kvals, err = h.GetNsValues(spec.DefaultKVNs)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, jsonError("get ns values failed: %v", err))
			return
		}
	} else {
		kvals, err = h.GetNsKindValues(spec.DefaultKVNs, kind)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, jsonError("get ns kind values failed: %v", err))
			return
		}
	}
	c.JSON(http.StatusOK, toSpecKVals(kvals))
}

func (h *Handler) PutKeys(c *gin.Context) {
	var specKVals spec.KeyVals
	if err := c.BindJSON(&specKVals); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonError("parse request body failed: %v", err))
		return
	}
	storeKVals := fromSpecKVals(specKVals)
	for _, kval := range storeKVals {
		if err := h.Set(kval.Key, kval.Value); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, jsonError("set key: %v failed: %v", kval.Key, err))
			return
		}
	}
	c.JSON(http.StatusOK, struct{}{})
}

func (h *Handler) PutKey(c *gin.Context) {
	var kval spec.KeyVal
	if err := c.BindJSON(&kval); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonError("parse request body failed: %v", err))
		return
	}
	storeKVal := fromSpecKVal(kval)
	if err := h.Set(storeKVal.Key, storeKVal.Value); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, jsonError("set key: %v failed: %v", storeKVal.Key, err))
		return
	}
	c.JSON(http.StatusOK, struct{}{})
}

func (h *Handler) GetKey(c *gin.Context) {
	rawKey := c.Param("fully_qualified_key_name")
	parts := strings.SplitN(rawKey, "/", 2)
	if len(parts) != 2 {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonError("invalid fully qualified key name: %v", rawKey))
		return
	}
	kind, name := parts[0], parts[1]
	val, err := h.Get(store.Key{
		Namespace: spec.DefaultKVNs,
		Kind:      kind,
		Name:      name,
	})
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonError("get key: %v failed: %v", rawKey, err))
		return
	}

	c.JSON(http.StatusOK, spec.KeyVal{
		Kind:  kind,
		Name:  name,
		Value: val.(string),
	})
}

func (h *Handler) GetSpecs(c *gin.Context) {}

func (h *Handler) GetSpec(c *gin.Context) {}

func (h *Handler) PutSpec(c *gin.Context) {}

func (h *Handler) PatchSpec(c *gin.Context) {}

func (h *Handler) PostDeployment(c *gin.Context) {}

func toSpecKVals(kvals store.KeyVals) spec.KeyVals {
	var specKVals spec.KeyVals
	for _, kval := range kvals {
		specKVals = append(specKVals, toSpecKVal(kval))
	}
	return specKVals
}

func toSpecKVal(kval store.KeyVal) spec.KeyVal {
	return spec.KeyVal{
		Kind:  kval.Kind,
		Name:  kval.Name,
		Value: kval.Value.(string),
	}
}

func fromSpecKVals(kvals spec.KeyVals) store.KeyVals {
	var storeKVals store.KeyVals
	for _, kval := range kvals {
		storeKVals = append(storeKVals, fromSpecKVal(kval))
	}
	return storeKVals
}

func fromSpecKVal(kval spec.KeyVal) store.KeyVal {
	return store.KeyVal{
		Key: store.Key{
			Namespace: spec.DefaultKVNs,
			Kind:      kval.Kind,
			Name:      kval.Name,
		},
		Value: nil,
	}
}

func jsonError(format string, a ...interface{}) string {
	sb := strings.Builder{}
	sb.WriteString(`{"error": "`)
	b, _ := json.Marshal(fmt.Sprintf(format, a...))
	sb.Write(b)
	sb.WriteString(`"}`)
	return sb.String()
}
