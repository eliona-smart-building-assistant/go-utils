//  This file is part of the eliona project.
//  Copyright © 2022 LEICOM iTEC AG. All Rights Reserved.
//  ______ _ _
// |  ____| (_)
// | |__  | |_  ___  _ __   __ _
// |  __| | | |/ _ \| '_ \ / _` |
// | |____| | | (_) | | | | (_| |
// |______|_|_|\___/|_| |_|\__,_|
//
//  THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR IMPLIED, INCLUDING
//  BUT NOT LIMITED  TO THE WARRANTIES OF MERCHANTABILITY, FITNESS FOR A PARTICULAR PURPOSE AND
//  NON INFRINGEMENT. IN NO EVENT SHALL THE AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM,
//  DAMAGES OR OTHER LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
//  OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE SOFTWARE.

package http

import (
	"bytes"
	"context"
	"crypto/tls"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/eliona-smart-building-assistant/go-utils/log"
	"github.com/gorilla/websocket"
)

// NewRequestWithBearer creates a new request for the given url. The url is authenticated with a barrier token.
func NewRequestWithBearer(url string, token string) (*http.Request, error) {
	return newRequestWithBearer(url, "GET", token)
}

// NewRequestWithApiKey creates a new request for the given url. The url is authenticated with a named api key.
func NewRequestWithApiKey(url string, key string, value string) (*http.Request, error) {
	return newRequestWithHeaders(url, "GET", map[string]string{key: value})
}

func NewDeleteRequestWithApiKey(url string, key string, value string) (*http.Request, error) {
	return newRequestWithHeaders(url, "DELETE", map[string]string{key: value})
}

func NewDeleteRequestWithApiKeyAndBody(url string, body any, key string, value string) (*http.Request, error) {
	return newRequestWithHeaderSecretAndBody(url, body, "DELETE", key, value)
}

func NewPatchRequestWithApiKey(url string, key string, value string) (*http.Request, error) {
	return newRequestWithHeaders(url, "PATCH", map[string]string{key: value})
}

func NewRequestWithHeaders(url string, headers map[string]string) (*http.Request, error) {
	return newRequestWithHeaders(url, "GET", headers)
}

func NewPostRequestWithHeaders(url string, body any, headers map[string]string) (*http.Request, error) {
	return newRequestWithHeadersAndBody(url, body, "POST", headers)
}

func encodeForm(form map[string][]string) string {
	values := url.Values{}
	for name, vs := range form {
		for _, v := range vs {
			values.Add(name, v)
		}
	}
	return values.Encode()
}

func NewPostFormRequestWithBasicAuth(url string, form map[string][]string, username string, password string) (*http.Request, error) {
	return NewPostFormRequestWithHeaders(url, form, map[string]string{"Authorization": "Basic " + base64.StdEncoding.EncodeToString([]byte(username+":"+password))})
}

func NewPostFormRequestWithHeaders(url string, form map[string][]string, headers map[string]string) (*http.Request, error) {
	request, err := newRequestWithBody(url, encodeForm(form), "application/x-www-form-urlencoded", "POST")
	if err != nil {
		log.Error("http", "error creating request %s: %v", url, err)
		return nil, err
	}
	for key, value := range headers {
		request.Header.Set(key, value)
	}
	return request, nil
}

// NewPostRequestWithBearer creates a new request for the given url. The url is authenticated with a barrier token.
func NewPostRequestWithBearer(url string, body any, token string) (*http.Request, error) {
	return newRequestWithBearerAndBody(url, body, "POST", token)
}

// NewPostRequestWithApiKey creates a new request for the given url. The url is authenticated with a named api key.
func NewPostRequestWithApiKey(url string, body any, key string, value string) (*http.Request, error) {
	return newRequestWithHeaderSecretAndBody(url, body, "POST", key, value)
}

// NewPutRequestWithBearer creates a new request for the given url. The url is authenticated with a barrier token.
func NewPutRequestWithBearer(url string, body any, token string) (*http.Request, error) {
	return newRequestWithBearerAndBody(url, body, "PUT", token)
}

// NewPutRequestWithApiKey creates a new request for the given url. The url is authenticated with a named api key.
func NewPutRequestWithApiKey(url string, body any, key string, value string) (*http.Request, error) {
	return newRequestWithHeaderSecretAndBody(url, body, "PUT", key, value)
}

func newRequestWithBearerAndBody(url string, body any, method string, token string) (*http.Request, error) {
	return newRequestWithHeaderSecretAndBody(url, body, method, "Authorization", "Bearer "+token)
}

func newRequestWithBearer(url string, method string, token string) (*http.Request, error) {
	return newRequestWithHeaders(url, method, map[string]string{"Authorization": "Bearer " + token})
}

// NewWebSocketConnectionWithApiKey creates a connection to a web socket. The url is authenticated with a named api key.
func NewWebSocketConnectionWithApiKey(url string, key string, value string) (*websocket.Conn, error) {
	dialer := websocket.Dialer{}
	header := http.Header{}
	header.Set(key, value)
	url = strings.Replace(url, "https://", "wss://", -1)
	url = strings.Replace(url, "http://", "ws://", -1)
	conn, _, err := dialer.Dial(url, header)
	if err != nil {
		log.Error("Websocket", "Error dialing websocket: %v", err)
		return nil, err
	}
	return conn, nil
}

func ListenWebSocketWithReconnect[T any](newWebSocket func() (*websocket.Conn, error), reconnectDelay time.Duration, objects chan T) error {
	var err error
	var conn *websocket.Conn
	defer func() {
		if conn != nil {
			_ = conn.Close()
		}
	}()
	for {
		conn, err = newWebSocket()
		if err != nil {
			log.Error("websocket", "Error creating web socket: %v", err)
			return err
		}
		err := ListenWebSocket(conn, objects)
		if err != nil {
			if closeError, ok := err.(*websocket.CloseError); ok {
				if closeError.Code == websocket.CloseAbnormalClosure {
					log.Debug("websocket", "Reconnecting web socket after abnormal closure: %v", err)
					time.Sleep(reconnectDelay)
					continue
				}
			}
		}
		return err
	}
}

// ListenWebSocket on a web socket connection and returns typed data
func ListenWebSocket[T any](conn *websocket.Conn, objects chan T) error {
	for {
		object, err := ReadWebSocket[T](conn)
		if err != nil || object == nil {
			return err
		}
		objects <- *object
	}
}

// ReadWebSocket reads one message from a web socket connection and returns typed data
func ReadWebSocket[T any](conn *websocket.Conn) (*T, error) {
	for {
		var object T
		tp, data, err := conn.ReadMessage()
		if err != nil {
			if closeError, ok := err.(*websocket.CloseError); ok {
				if closeError.Code == websocket.CloseNormalClosure {
					return nil, nil
				} else {
					log.Error("websocket", "Error reading web socket: %v", closeError)
				}
			} else {
				log.Error("websocket", "Error reading web socket: %v", err)
			}
			return nil, err
		}
		if tp == websocket.TextMessage {
			err := json.Unmarshal(data, &object)
			if err != nil {
				log.Error("websocket", "Error unmarshalling message from web socket: %v", err)
				return nil, err
			}
			return &object, nil
		}
	}
}

func newRequestWithHeaderSecretAndBody(url string, body any, method string, key string, value string) (*http.Request, error) {

	// Create a new request
	request, err := newRequestWithBody(url, body, "application/json", method)
	if err != nil {
		log.Error("Http", "Error creating request %s: %v", url, err)
		return nil, err
	}

	request.Header.Set(key, value)
	return request, nil
}

func newRequestWithHeadersAndBody(url string, body any, method string, headers map[string]string) (*http.Request, error) {

	// Create a new request
	request, err := newRequestWithBody(url, body, "application/json", method)
	if err != nil {
		log.Error("Http", "Error creating request %s: %v", url, err)
		return nil, err
	}

	for key, value := range headers {
		request.Header.Set(key, value)
	}
	return request, nil
}

func newRequestWithHeaders(url string, method string, headers map[string]string) (*http.Request, error) {
	request, err := newRequest(url, method)
	if err != nil {
		log.Error("Http", "Error creating request %s: %v", url, err)
		return nil, err
	}

	for key, value := range headers {
		request.Header.Set(key, value)
	}
	return request, nil
}

// NewRequest creates a new request for the given url. The url have to provide free access without any
// authentication. For authentication use other functions like NewGetRequestWithBarrier.
func NewRequest(url string) (*http.Request, error) {
	return newRequest(url, "GET")
}

// NewPostRequest creates a new request for the given url and the body as payload. The url have to provide free access without any
// authentication. For authentication use other functions like NewPostRequestWithBearer.
func NewPostRequest(url string, body any) (*http.Request, error) {
	return newRequestWithBody(url, body, "application/json", "POST")
}

// NewPutRequest creates a new request for the given url and the body as payload. The url have to provide free access without any
// authentication. For authentication use other functions like NewPutRequestWithBearer.
func NewPutRequest(url string, body any) (*http.Request, error) {
	return newRequestWithBody(url, body, "application/json", "PUT")
}

func newRequestWithBody(url string, body any, contentType string, method string) (*http.Request, error) {

	// Create payload if used
	var payload []byte
	var err error
	if contentType == "application/json" {
		payload, err = json.Marshal(body)
		if err != nil {
			log.Error("Kafka", "Failed to marshal body: %s", err.Error())
			return nil, err
		}
	} else {
		payload = []byte(body.(string))
	}

	// Create a new request with payload if used
	request, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
	if err != nil {
		log.Error("Http", "Error creating request %s: %v", url, err)
		return nil, err
	}

	request.Header.Set("Content-Type", contentType)
	return request, nil
}

func newRequest(url string, method string) (*http.Request, error) {

	// Create a new request with payload if used
	request, err := http.NewRequest(method, url, nil)
	if err != nil {
		log.Error("Http", "Error creating request %s: %v", url, err)
		return nil, err
	}
	return request, nil
}

// Read returns the response data converted to a corresponding structure
func Read[T any](request *http.Request, timeout time.Duration, checkCertificate bool) (T, error) {
	body, _, err := ReadWithStatusCode[T](request, timeout, checkCertificate)
	if err != nil {
		return body, err
	}
	return body, nil
}

// ReadWithStatusCode returns the response data converted to a corresponding structure
func ReadWithStatusCode[T any](request *http.Request, timeout time.Duration, checkCertificate bool) (T, int, error) {
	var value T

	payload, statusCode, err := DoWithStatusCode(request, timeout, checkCertificate)
	if err != nil {
		return value, statusCode, fmt.Errorf("do with status code: %v", err)
	}

	if len(payload) == 0 {
		return value, statusCode, nil
	}

	if json.Valid(payload) {
		err := json.Unmarshal(payload, &value)
		if err != nil {
			log.Error("Http", "Unmarshal error: %v (%s)", err, string(payload))
			return value, statusCode, fmt.Errorf("unmarshaling: %v", err)
		}
		return value, statusCode, nil
	} else if _, ok := interface{}(value).(string); ok {
		return any(string(payload)).(T), statusCode, nil
	} else {
		return value, statusCode, fmt.Errorf("invalid payload format: %v", string(payload))
	}
}

// Do return the payload returned from the request
func Do(request *http.Request, timeout time.Duration, checkCertificate bool) ([]byte, error) {
	body, statusCode, err := DoWithStatusCode(request, timeout, checkCertificate)
	if err != nil {
		return body, err
	}
	if statusCode < 300 {
		return body, nil
	} else {
		log.Error("Http", "Error request code %d for request to %s.", statusCode, request.URL)
		return nil, fmt.Errorf("error request code %d for request to %s", statusCode, request.URL)
	}
}

// DoWithStatusCode return the body and the code
func DoWithStatusCode(request *http.Request, timeout time.Duration, checkCertificate bool) ([]byte, int, error) {

	// creates a http client with timeout and tsl security configuration
	httpClient := http.Client{
		Timeout: timeout,
		Transport: &http.Transport{
			TLSClientConfig: &tls.Config{InsecureSkipVerify: !checkCertificate},
		},
	}

	// start the request
	response, err := httpClient.Do(request)
	if err != nil {
		log.Error("Http", "Error request to %s: %v", request.URL, err)
		return nil, 0, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error("Http", "Error closing request for %s: %v", request.URL, err)
		}
	}(response.Body)

	// read the complete payload
	body, err := io.ReadAll(response.Body)
	if err != nil {
		log.Error("Http", "Error body from %s: %v", request.URL, err)
		return nil, response.StatusCode, err
	}

	return body, response.StatusCode, nil
}

// ListenApiWithOs starts an API server and listen for API requests
func ListenApiWithOs(server *http.Server) {

	// channel to get os signals
	osSignal := make(chan os.Signal, 1)
	defer close(osSignal)
	signal.Notify(osSignal, syscall.SIGINT, syscall.SIGTERM)

	//  start server
	closeChan := make(chan bool)
	go func() {
		err := server.ListenAndServe()
		closeChan <- true
		if err != nil {
			log.Fatal("main", "Error in API Server: %v", err)
		}
	}()

	// wait for server close or os signals
	select {
	case <-closeChan:
		return
	case <-osSignal:
		err := server.Shutdown(context.Background())
		if err != nil {
			log.Error("api", "Error in API Server: %v", err)
			return
		}
		return
	}

}

type CORSEnabledHandler struct {
	handler http.Handler
}

func NewCORSEnabledHandler(handler http.Handler) CORSEnabledHandler {
	return CORSEnabledHandler{handler: handler}
}

func (h CORSEnabledHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if r.Method == "OPTIONS" {
		// Set CORS headers and allow the requested methods.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Api-Key")

		// Respond with 200 OK status.
		w.WriteHeader(http.StatusOK)
		return
	}
	w.Header().Set("Access-Control-Allow-Origin", "*")
	h.handler.ServeHTTP(w, r)
}
