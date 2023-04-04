package envs

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"reflect"
	"strconv"
	"strings"
	"time"

	"github.com/ghodss/yaml"
	"github.com/gin-gonic/gin"
	goyaml "gopkg.in/yaml.v2"
	"tespkg.in/envs/pkg/kvs"
	"tespkg.in/envs/pkg/store"
)

type Handler struct {
	s    store.Store
	cred *kvs.Creds
}

func NewHandler(s store.Store, creds *kvs.Creds) *Handler {
	return &Handler{
		s:    s,
		cred: creds,
	}
}

func (h *Handler) listByPrefix(c *gin.Context, prefix store.Key, trimPrefix bool) {
	keyVals, err := h.s.ListByPrefix(prefix)
	if err != nil {
		if errors.Is(err, store.ErrNotFound) {
			c.AbortWithStatusJSON(http.StatusNotFound, jsonErrorf("prefix %v not found", prefix))
			return
		}
		c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("get by prefix: %v failed: %v", prefix, err))
		return
	}
	res := make(kvs.KeyVals, 0, len(keyVals))
	for _, v := range keyVals {
		val, ok := v.Value.(string)
		if !ok {
			c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("invalid value type: %s for %s", reflect.TypeOf(val), val))
			return
		}
		kName := v.Name
		if trimPrefix {
			kName = strings.TrimPrefix(kName, prefix.Name)
		}
		res = append(res, kvs.KeyVal{
			Key: kvs.Key{
				Kind: prefix.Kind,
				Name: kName,
			},
			Value: val,
		})
	}
	marshaled, err := json.Marshal(res)
	if err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("marshal KeyVals failed: %v", res))
		return
	}
	c.JSON(http.StatusOK, kvs.KeyVal{
		Key: kvs.Key{
			Kind: prefix.Kind,
			Name: prefix.Name,
		},
		Value: string(marshaled),
	})
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
	isPrefix := c.Query(`is_prefix`) == `true`
	trimPrefix := c.Query(`trim_prefix`) == `true`
	key := store.Key{
		Namespace: ns,
		Kind:      kind,
		Name:      name,
	}
	if isPrefix {
		h.listByPrefix(c, key, trimPrefix)
		return
	}

	val, err := h.s.Get(key)
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
		kvals, err = h.s.GetNsValues(ns)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("get ns values failed: %v", err))
			return
		}
	} else {
		kvals, err = h.s.GetNsKindValues(ns, kind)
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
	if kval.Kind == kvs.EnvkKind && getIsPlaintext(c) {
		ciphertext, err := h.cred.Encrypt(kval.Value)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("encrypt key: %v failed: %v", kval.Name, err))
			return
		}
		kval.Value = ciphertext
	}

	ns := getNamespace(c)
	storeKVal, err := kv2storeKV(ns, kval, getIsJson(c))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("parse key: %v failed: %v", kval.Name, err))
		return
	}
	if err := h.s.Set(storeKVal.Key, storeKVal.Value); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("set key: %v failed: %v", storeKVal.Key, err))
		return
	}
	c.JSON(http.StatusOK, struct{}{})
}

type EnvKeyVal struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

func (h *Handler) PutEnvKey(c *gin.Context) {
	var envKVal EnvKeyVal
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
	storeKVal, err := kv2storeKV(ns, kval, getIsJson(c))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("parse key: %v failed: %v", kval.Name, err))
		return
	}
	if err := h.s.Set(storeKVal.Key, storeKVal.Value); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("set key: %v failed: %v", storeKVal.Key, err))
		return
	}
	c.JSON(http.StatusOK, struct{}{})
}

func (h *Handler) PutEnvKeys(c *gin.Context) {
	var envKeyVals []EnvKeyVal
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
	storeKVals, err := kvs2storeKVs(ns, kvals, getIsJson(c))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("parse keys failed: %v", err))
		return
	}
	for _, kval := range storeKVals {
		if err := h.s.Set(kval.Key, kval.Value); err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("set key: %v failed: %v", kval.Key, err))
			return
		}
	}
	c.JSON(http.StatusOK, struct{}{})
}

func (h *Handler) ExportEnvKVS(c *gin.Context) {
	ns := getNamespace(c)
	storeKVals, err := h.s.GetNsKindValues(ns, kvs.EnvKind)
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
	storeKVals, err := kvs2storeKVs(ns, kvals, getIsJson(c))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("parse keys failed: %v", err))
		return
	}
	for _, kval := range storeKVals {
		if err := h.s.Set(kval.Key, kval.Value); err != nil {
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
	storeKVal, err := kv2storeKV(ns, kval, getIsJson(c))
	if err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("parse key: %v failed: %v", kval.Name, err))
		return
	}
	if err := h.s.Set(storeKVal.Key, storeKVal.Value); err != nil {
		c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("set key: %v failed: %v", storeKVal.Key, err))
		return
	}
	c.JSON(http.StatusOK, struct{}{})
}

type SecretRequest struct {
	Plaintexts []string `json:"plaintexts"`
}

type Secrets struct {
	Plaintexts  []string `json:"plaintexts"`
	Ciphertexts []string `json:"ciphertexts"`
}

func (h *Handler) GenerateSecret(c *gin.Context) {
	var req SecretRequest
	if err := c.BindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("parse request body failed: %v", err))
		return
	}

	var ciphers []string
	for _, txt := range req.Plaintexts {
		ciphertext, err := h.cred.Encrypt(txt)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("generate secret failed: %v", err))
			return
		}
		ciphers = append(ciphers, ciphertext)
	}
	result := &Secrets{
		Plaintexts:  req.Plaintexts,
		Ciphertexts: ciphers,
	}
	c.JSON(http.StatusOK, result)
}

func (h *Handler) VerifySecret(c *gin.Context) {
	var req Secrets
	if err := c.BindJSON(&req); err != nil {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("parse request body failed: %v", err))
		return
	}

	if len(req.Ciphertexts) != len(req.Plaintexts) {
		c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("invalid request"))
		return
	}
	for i, ciphertext := range req.Ciphertexts {
		got, err := h.cred.Decrypt(ciphertext)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, jsonErrorf("decrypt secret failed"))
			return
		}
		if got != req.Plaintexts[i] {
			c.AbortWithStatusJSON(http.StatusBadRequest, jsonErrorf("invalid secret pair at index %v", i))
			return
		}
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

func kvs2storeKVs(ns string, kvals kvs.KeyVals, isJson bool) (store.KeyVals, error) {
	storeKVals := store.KeyVals{}
	for _, kval := range kvals {
		storeKVal, err := kv2storeKV(ns, kval, isJson)
		if err != nil {
			return nil, err
		}
		storeKVals = append(storeKVals, storeKVal)
	}
	return storeKVals, nil
}

func kv2storeKV(ns string, kval kvs.KeyVal, isJson bool) (store.KeyVal, error) {
	value := kval.Value
	if isJson {
		parsedValue, err := parseJsonValue(value)
		if err != nil {
			return store.KeyVal{}, err
		}
		value = parsedValue
	}
	return store.KeyVal{
		Key: store.Key{
			Namespace: ns,
			Kind:      kval.Kind,
			Name:      kval.Name,
		},
		Value: value,
	}, nil
}

func parseJsonValue(value string) (string, error) {
	// try top-level json array first
	var am []interface{}
	if err := json.Unmarshal([]byte(value), &am); err == nil {
		finalizedValue, _ := json.Marshal(&am)
		return string(finalizedValue), nil
	}
	// try top-level json object
	m := make(map[string]interface{})
	if err := json.Unmarshal([]byte(value), &m); err != nil {
		return "", fmt.Errorf("invalid json, err: %v", err)
	}
	finalizedValue, _ := json.Marshal(&m)
	return string(finalizedValue), nil
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

func getIsJson(c *gin.Context) bool {
	v := c.Query("json")
	return v == "true"
}

func getIsPlaintext(c *gin.Context) bool {
	v := c.Query("plain")
	return v == "true"
}
