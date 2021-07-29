package api

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"

	"tespkg.in/envs/pkg/kvs"
	"tespkg.in/kit/log"
)

const (
	// HTTPAddrEnvName defines an environment variable name which sets
	// the HTTP address if there is no -http-addr specified.
	HTTPAddrEnvName = "ENVS_HTTP_ADDR"
)

const (
	keyActSet = `set`
	keyActGet = `get`
)

var (
	keyActionCnt = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: `enva_key_actiono_counter`,
			Help: `counter for enva key action`,
		},
		[]string{`key`, `action`},
	)
)

// Config is used to configure the creation of a client
type Config struct {
	// Address is the address of the envs
	Address string

	// Scheme is the URI scheme for the envs
	Scheme string

	Namespace string

	// HTTPClient is the client to use. Default will be
	// used if not provided.
	HTTPClient *http.Client
}

// defaultConfig returns the default configuration for the client, using the
// given function to make the transport.
func defaultConfig() *Config {
	config := &Config{
		Address: "127.0.0.1:9112",
		Scheme:  "http",
	}

	if addr := os.Getenv(HTTPAddrEnvName); addr != "" {
		config.Address = addr
	}

	return config
}

// Client provides a client to the envs APIs
type Client struct {
	config Config
}

// NewClient returns a new client
func NewClient(config *Config) (*Client, error) {
	// bootstrap the config
	defConfig := defaultConfig()

	if len(config.Address) == 0 {
		config.Address = defConfig.Address
	}

	if len(config.Scheme) == 0 {
		config.Scheme = defConfig.Scheme
	}

	if config.HTTPClient == nil {
		config.HTTPClient = http.DefaultClient
	}

	parts := strings.SplitN(config.Address, "://", 2)
	if len(parts) == 2 {
		switch parts[0] {
		case "http":
			config.Scheme = "http"
		case "https":
			config.Scheme = "https"
		default:
			return nil, fmt.Errorf("unknown protocol scheme: %s", parts[0])
		}
		config.Address = parts[1]
	}

	return &Client{config: *config}, nil
}

// request is used to help build up a request
type request struct {
	config *Config
	method string
	url    *url.URL
	params url.Values
	body   io.Reader
	header http.Header
	obj    interface{}
	ctx    context.Context
}

// decodeBody is used to JSON decode a body
func decodeBody(resp *http.Response, out interface{}) error {
	dec := json.NewDecoder(resp.Body)
	return dec.Decode(out)
}

// encodeBody is used to encode a request body
func encodeBody(obj interface{}) (io.Reader, error) {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	if err := enc.Encode(obj); err != nil {
		return nil, err
	}
	return buf, nil
}

// responseCodeError consumes the rest of the body, closes
// the body stream and generates an error indicating the status code was unexpected.
func responseCodeError(resp *http.Response) error {
	bs, _ := ioutil.ReadAll(resp.Body)
	resp.Body.Close()
	message := resp.Status
	if len(bs) > 0 {
		var errBody = struct {
			Error string `json:"error"`
		}{}
		_ = json.Unmarshal(bs, &errBody)
		message += " " + errBody.Error
	}
	return fmt.Errorf("unexpected response code: %d (%s)", resp.StatusCode, message)
}

// requireOK is used to wrap doRequest and check for a 200
func requireOK(resp *http.Response, e error) (*http.Response, error) {
	if e != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return nil, e
	}
	if resp.StatusCode != 200 {
		return nil, responseCodeError(resp)
	}
	return resp, nil
}

func requireNotFoundOrOK(resp *http.Response, e error) (bool, *http.Response, error) {
	if e != nil {
		if resp != nil {
			resp.Body.Close()
		}
		return false, nil, e
	}
	switch resp.StatusCode {
	case 200:
		return true, resp, nil
	case 404:
		return false, resp, nil
	default:
		return false, nil, responseCodeError(resp)
	}
}

// toHTTP converts the request to an HTTP request
func (r *request) toHTTP() (*http.Request, error) {
	// Encode the query parameters
	r.url.RawQuery = r.params.Encode()

	// Check if we should encode the body
	if r.body == nil && r.obj != nil {
		b, err := encodeBody(r.obj)
		if err != nil {
			return nil, err
		}
		r.body = b
	}

	// Create the HTTP request
	req, err := http.NewRequest(r.method, r.url.RequestURI(), r.body)
	if err != nil {
		return nil, err
	}

	req.URL.Host = r.url.Host
	req.URL.Scheme = r.url.Scheme
	req.Host = r.url.Host
	req.Header = r.header

	if r.ctx != nil {
		return req.WithContext(r.ctx), nil
	}

	return req, nil
}

// newRequest is used to create a new request
func (c *Client) newRequest(method, path string, params map[string]string) *request {
	r := &request{
		config: &c.config,
		method: method,
		url: &url.URL{
			Scheme: c.config.Scheme,
			Host:   c.config.Address,
			Path:   path,
		},
		params: map[string][]string{
			"ns": {c.config.Namespace},
		},
		header: make(http.Header),
	}
	if params != nil {
		for k, v := range params {
			r.params[k] = append(r.params[k], v)
		}
	}
	return r
}

// doRequest runs a request with our client
func (c *Client) doRequest(r *request) (*http.Response, error) {
	req, err := r.toHTTP()
	if err != nil {
		return nil, err
	}
	resp, err := c.config.HTTPClient.Do(req)
	return resp, err
}

// Query is used to do a GET request against an endpoint
// and deserialize the response into an interface.
func (c *Client) query(endpoint string, params map[string]string, out interface{}) error {
	r := c.newRequest(http.MethodGet, endpoint, params)
	found, resp, err := requireNotFoundOrOK(c.doRequest(r))
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if !found {
		return kvs.ErrNotFound
	}

	if err := decodeBody(resp, out); err != nil {
		return err
	}
	return nil
}

func (c *Client) Get(key kvs.Key, isPrefix bool) (string, error) {
	kval := kvs.KeyVal{}
	params := make(map[string]string, 1)
	if isPrefix {
		params[`is_prefix`] = `true`
		params[`trim_prefix`] = `true`
		if !strings.HasSuffix(key.Name, "/") {
			// by convention, we use "/" to concat key and sub-key in both Set and Get side.
			// adding a "/" to the tail of the key, to make the replied top-level keys are been trimmed with "/".
			// e.g, the higher-level caller publish/set foor/bar = alice when set,
			// by convention, during the get, by specifying prefix = foo, it will get foo = {"bar": "alice"} as the response.
			key.Name = key.Name + "/"
		}
	}
	keyInPath := fmt.Sprintf("%s/%s", key.Kind, key.Name)
	endpoint := fmt.Sprintf("/key/%s", keyInPath)

	var err error
	defer func() {
		log.Debugf("Get value of key %v, is_prefix: %v, value: %v, length: %v, err: %v", key, isPrefix, kval.Value, len(kval.Value), err)
	}()

	err = c.query(endpoint, params, &kval)
	if err != nil {
		return "", err
	}
	keyActionCnt.WithLabelValues(key.Name, keyActGet).Inc()
	return kval.Value, nil
}

func (c *Client) Set(key kvs.Key, value string) error {
	kval := kvs.KeyVal{
		Key:   key,
		Value: value,
	}
	r := c.newRequest(http.MethodPut, "/key", nil)
	r.obj = kval

	var resp *http.Response
	var err error

	defer func() {
		log.Debugf("Put key %v with value: %v, length: %v, err: %v", key, value, len(value), err)
	}()

	resp, err = requireOK(c.doRequest(r))
	if err != nil {
		return err
	}
	resp.Body.Close()

	keyActionCnt.WithLabelValues(key.Name, keyActSet).Inc()

	return nil
}
