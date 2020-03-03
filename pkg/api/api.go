package api

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"tespkg.in/envs/pkg/kvs"
)

const (
	// HTTPAddrEnvName defines an environment variable name which sets
	// the HTTP address if there is no -http-addr specified.
	HTTPAddrEnvName = "ENVS_HTTP_ADDR"
)

// Config is used to configure the creation of a client
type Config struct {
	// Address is the address of the envs
	Address string

	// Scheme is the URI scheme for the envs
	Scheme string

	// HttpClient is the client to use. Default will be
	// used if not provided.
	HttpClient *http.Client
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

	if config.HttpClient == nil {
		config.HttpClient = http.DefaultClient
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
func (c *Client) newRequest(method, path string) *request {
	r := &request{
		config: &c.config,
		method: method,
		url: &url.URL{
			Scheme: c.config.Scheme,
			Host:   c.config.Address,
			Path:   path,
		},
		params: make(map[string][]string),
		header: make(http.Header),
	}
	return r
}

// doRequest runs a request with our client
func (c *Client) doRequest(r *request) (time.Duration, *http.Response, error) {
	req, err := r.toHTTP()
	if err != nil {
		return 0, nil, err
	}
	start := time.Now()
	resp, err := c.config.HttpClient.Do(req)
	diff := time.Since(start)
	return diff, resp, err
}

// Query is used to do a GET request against an endpoint
// and deserialize the response into an interface.
func (c *Client) query(endpoint string, out interface{}) error {
	r := c.newRequest("GET", endpoint)
	_, resp, err := c.doRequest(r)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		bs, _ := ioutil.ReadAll(resp.Body)
		message := resp.Status
		if len(bs) > 0 {
			var errBody = struct {
				Error string `json:"error"`
			}{}
			_ = json.Unmarshal(bs, &errBody)
			message += " " + errBody.Error
		}
		return errors.New(message)
	}

	if err := decodeBody(resp, out); err != nil {
		return err
	}
	return nil
}

func (c *Client) Get(key kvs.Key) (string, error) {
	kval := kvs.KeyVal{}
	keyInPath := fmt.Sprintf("%s/%s", key.Kind, key.Name)
	endpoint := fmt.Sprintf("/key/%s", keyInPath)
	if err := c.query(endpoint, &kval); err != nil {
		return "", err
	}
	return kval.Value, nil
}
