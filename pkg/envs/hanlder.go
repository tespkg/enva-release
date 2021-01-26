package envs

import (
	"bytes"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	goyaml "gopkg.in/yaml.v2"
	"tespkg.in/envs/pkg/addons"
	"tespkg.in/envs/pkg/kvs"
	"tespkg.in/envs/pkg/store"
)

type Handler struct {
	store.Store
}

func NewHandler(s store.Store) *Handler {
	return &Handler{
		Store: s,
	}
}

func (h *Handler) GetKey(c *gin.Context) {
	ns := getNamespace(c)
	rawKey := strings.TrimPrefix(c.Param("fully_qualified_key_name"), "/")
	parts := strings.SplitN(rawKey, "/", 2)
	if len(parts) != 2 {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("invalid fully qualified key name: %v", rawKey))
		return
	}
	kind, name := parts[0], parts[1]
	val, err := h.Get(store.Key{
		Namespace: ns,
		Kind:      kind,
		Name:      name,
	})
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, jsonErrorf("%v not found", rawKey))
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("get key: %v failed: %v", rawKey, err))
		return
	}

	c.JSON(http.StatusOK, kvs.KeyVal{
		Key: kvs.Key{
			Kind: kind,
			Name: name,
		},
		Value: val.(string),
	})
}

func (h *Handler) GetKeys(c *gin.Context) {
	ns := getNamespace(c)
	kind := c.Query("kind")

	var kvals store.KeyVals
	var err error
	if kind == "" {
		kvals, err = h.GetNsValues(ns)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("get ns values failed: %v", err))
			return
		}
	} else {
		kvals, err = h.GetNsKindValues(ns, kind)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("get ns kind values failed: %v", err))
			return
		}
	}
	c.JSON(http.StatusOK, storeKVs2kvs(kvals))
}

func (h *Handler) PutKey(c *gin.Context) {
	var kval kvs.KeyVal
	if err := c.BindJSON(&kval); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("parse request body failed: %v", err))
		return
	}

	ns := getNamespace(c)
	storeKVal := kv2storeKV(ns, kval)
	if err := h.Set(storeKVal.Key, storeKVal.Value); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("set key: %v failed: %v", storeKVal.Key, err))
		return
	}
	c.JSON(http.StatusOK, struct{}{})
}

func (h *Handler) PutEnvKey(c *gin.Context) {
	var envKVal kvs.EnvKeyVal
	if err := c.BindJSON(&envKVal); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("parse request body failed: %v", err))
		return
	}

	ns := getNamespace(c)
	kval := kvs.KeyVal{
		Key: kvs.Key{
			Name: envKVal.Name,
			Kind: kvs.EnvKind,
		},
		Value: envKVal.Value,
	}
	storeKVal := kv2storeKV(ns, kval)
	if err := h.Set(storeKVal.Key, storeKVal.Value); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("set key: %v failed: %v", storeKVal.Key, err))
		return
	}
	c.JSON(http.StatusOK, struct{}{})
}

func (h *Handler) PutEnvKeys(c *gin.Context) {
	var envKeyVals []kvs.EnvKeyVal
	if err := c.BindJSON(&envKeyVals); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("parse request body failed: %v", err))
		return
	}

	kvals := kvs.KeyVals{}
	for _, envKeyVal := range envKeyVals {
		kvals = append(kvals, kvs.KeyVal{
			Key: kvs.Key{
				Kind: kvs.EnvKind,
				Name: envKeyVal.Name,
			},
			Value: envKeyVal.Value,
		})
	}
	ns := getNamespace(c)
	storeKVals := kvs2storeKVs(ns, kvals)
	for _, kval := range storeKVals {
		if err := h.Set(kval.Key, kval.Value); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("set key: %v failed: %v", kval.Key, err))
			return
		}
	}
	c.JSON(http.StatusOK, struct{}{})
}

func (h *Handler) ExportEnvKVS(c *gin.Context) {
	ns := getNamespace(c)
	storeKVals, err := h.GetNsKindValues(ns, kvs.EnvKind)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("iterate env kind keys failed: %v", err))
		return
	}

	kvsKVals := storeKVs2kvs(storeKVals)
	out, err := yaml.Marshal(kvsKVals)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("marshal kvals failed: %v", err))
		return
	}

	c.Header("Content-Disposition", fmt.Sprintf(`attachment; filename="kvs-%s.yaml"`, time.Now().Format("2006-01-02")))
	c.Header("Content-Type", "application/json")
	c.Header("Content-Length", strconv.Itoa(len(out)))

	if _, err := io.Copy(c.Writer, bytes.NewBuffer(out)); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("write file failed: %v", err))
		return
	}
}

func (h *Handler) ImportEnvKVS(c *gin.Context) {
	multiForm, err := c.MultipartForm()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("invalid multipart form: %v", err))
		return
	}

	var filenames []string
	var fds []multipart.File
	defer func() {
		for _, fd := range fds {
			fd.Close()
		}
	}()

	for _, fs := range multiForm.File {
		for _, f := range fs {
			filename := f.Filename
			filenames = append(filenames, filename)
			fd, err := f.Open()
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("open file: %v failed: %v", filename, err))
				return
			}
			fds = append(fds, fd)
		}
	}

	if len(filenames) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("no file was found"))
		return
	}
	if len(filenames) > 1 {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("multiple files unsupported yet"))
		return
	}

	dec := goyaml.NewDecoder(fds[0])
	var res [][]byte
	for {
		var value interface{}
		err := dec.Decode(&value)
		if err == io.EOF {
			break
		}
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("invalid file content: %v", err))
			return
		}
		valueBytes, err := goyaml.Marshal(value)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("marshal file content: %v", err))
			return
		}
		res = append(res, valueBytes)
	}

	kvals := kvs.KeyVals{}
	for _, out := range res {
		vals := kvs.KeyVals{}
		if err := yaml.Unmarshal(out, &vals); err != nil {
			c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("invalid file content: %v", err))
			return
		}
		kvals = append(kvals, vals...)
	}

	ns := getNamespace(c)
	storeKVals := kvs2storeKVs(ns, kvals)
	for _, kval := range storeKVals {
		if err := h.Set(kval.Key, kval.Value); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("set key %v failed: %v", kval.Key, err))
			return
		}
	}

	c.JSON(http.StatusOK, struct{}{})
}

func (h *Handler) PutEnvfKey(c *gin.Context) {
	name := c.Query("name")
	if name == "" {
		c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("empty name in query string"))
		return
	}

	multiForm, err := c.MultipartForm()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("invalid multipart form: %v", err))
		return
	}

	var filenames []string
	var fds []multipart.File
	defer func() {
		for _, fd := range fds {
			fd.Close()
		}
	}()

	for _, fs := range multiForm.File {
		for _, f := range fs {
			filename := f.Filename
			filenames = append(filenames, filename)
			fd, err := f.Open()
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("open file: %v failed: %v", filename, err))
				return
			}
			fds = append(fds, fd)
		}
	}

	if len(filenames) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("no file was found"))
		return
	}
	if len(filenames) > 1 {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("multiple files unsupported yet"))
		return
	}
	data, err := ioutil.ReadAll(fds[0])
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("read envf content failed: %v", err))
		return
	}

	kval := kvs.KeyVal{
		Key: kvs.Key{
			Kind: kvs.EnvfKind,
			Name: name,
		},
		Value: string(data),
	}
	ns := getNamespace(c)
	storeKVal := kv2storeKV(ns, kval)
	if err := h.Set(storeKVal.Key, storeKVal.Value); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("set key: %v failed: %v", storeKVal.Key, err))
		return
	}
	c.JSON(http.StatusOK, struct{}{})
}

func (h *Handler) OAuthRegistration(c *gin.Context) {
	multiForm, err := c.MultipartForm()
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("invalid multipart form: %v", err))
		return
	}

	var filenames []string
	var fds []multipart.File
	defer func() {
		for _, fd := range fds {
			fd.Close()
		}
	}()

	for _, fs := range multiForm.File {
		for _, f := range fs {
			filename := f.Filename
			filenames = append(filenames, filename)
			fd, err := f.Open()
			if err != nil {
				c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("open file: %v failed: %v", filename, err))
				return
			}
			fds = append(fds, fd)
		}
	}

	if len(filenames) == 0 {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("no file was found"))
		return
	}
	if len(filenames) > 1 {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("multiple files unsupported yet"))
		return
	}

	// Render the imported file content if there is any env key need to render from envs
	ns := getNamespace(c)
	oidcr := addons.NewOidcRegister(ns, h.Store)
	out := bytes.Buffer{}
	if err := kvs.Render(oidcr, fds[0], &out); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("invalid file content: %v", err))
		return
	}

	req := addons.OAuthRegistrationReq{}
	if err := yaml.Unmarshal(out.Bytes(), &req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("invalid file content: %v", err))
		return
	}

	provider := req.ProviderConfig
	if provider.Issuer == "" {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("empty oidc issuer"))
		return
	}

	if err := addons.RegisterOAuthClients(oidcr, req.ProviderConfig, req.Requests); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("register OAuth client failed: %v", err))
		return
	}

	c.JSON(http.StatusOK, struct{}{})
}

func storeKVs2kvs(kvals store.KeyVals) kvs.KeyVals {
	specKVals := kvs.KeyVals{}
	for _, kval := range kvals {
		specKVals = append(specKVals, storeKV2kv(kval))
	}
	return specKVals
}

func storeKV2kv(kval store.KeyVal) kvs.KeyVal {
	return kvs.KeyVal{
		Key: kvs.Key{
			Kind: kval.Kind,
			Name: kval.Name,
		},
		Value: kval.Value.(string),
	}
}

func kvs2storeKVs(ns string, kvals kvs.KeyVals) store.KeyVals {
	storeKVals := store.KeyVals{}
	for _, kval := range kvals {
		storeKVals = append(storeKVals, kv2storeKV(ns, kval))
	}
	return storeKVals
}

func kv2storeKV(ns string, kval kvs.KeyVal) store.KeyVal {
	return store.KeyVal{
		Key: store.Key{
			Namespace: ns,
			Kind:      kval.Kind,
			Name:      kval.Name,
		},
		Value: kval.Value,
	}
}

func jsonErrorf(format string, a ...interface{}) interface{} {
	return struct {
		Error string `json:"error"`
	}{
		Error: fmt.Sprintf(format, a...),
	}
}

func getNamespace(c *gin.Context) string {
	ns := c.Query("ns")
	if ns == "" {
		ns = store.DefaultKVNs
	}
	return ns
}
