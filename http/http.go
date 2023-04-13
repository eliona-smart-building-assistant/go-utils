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
	"encoding/json"
	"fmt"
	"github.com/eliona-smart-building-assistant/go-utils/log"
	"github.com/gorilla/websocket"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"
)

// NewRequestWithBearer creates a new request for the given url. The url is authenticated with a barrier token.
func NewRequestWithBearer(url string, token string) (*http.Request, error) {
	return newRequestWithBearer(url, "GET", token)
}

// NewRequestWithApiKey creates a new request for the given url. The url is authenticated with a named api key.
func NewRequestWithApiKey(url string, key string, value string) (*http.Request, error) {
	return newRequestWithHeaderSecret(url, "GET", key, value)
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
	return newRequestWithHeaderSecret(url, method, "Authorization", "Bearer "+token)
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

func ListenWebSocketWithReconnect[T any](newWebSocket func() (*websocket.Conn, error), reconnectDelay time.Duration, objects chan T) {
	defer close(objects)
	var err error
	var conn *websocket.Conn
	defer conn.Close()
	for {
		conn, err = newWebSocket()
		if err != nil {
			break
		}
		object, err := ReadWebSocket[T](conn)
		if err != nil {
			if closeError, ok := err.(*websocket.CloseError); ok {
				if closeError.Code == websocket.CloseAbnormalClosure {
					log.Info("websocket", "Reconnecting web socket after abnormal closure: %v", err)
					time.Sleep(reconnectDelay)
					continue
				}
			}
			break
		}
		objects <- object
	}
}

// ListenWebSocket on a web socket connection and returns typed data
func ListenWebSocket[T any](conn *websocket.Conn, objects chan T) {
	defer close(objects)
	for {
		object, err := ReadWebSocket[T](conn)
		if err != nil {
			break
		}
		objects <- object
	}
}

// ReadWebSocket reads one message from a web socket connection and returns typed data
func ReadWebSocket[T any](conn *websocket.Conn) (T, error) {
	for {
		var object T
		tp, data, err := conn.ReadMessage()
		if err != nil {
			if closeError, ok := err.(*websocket.CloseError); ok {
				if closeError.Code == websocket.CloseAbnormalClosure {
					return object, err
				}
			}
			log.Error("websocket", "Error reading web socket: %v", err)
			return object, err
		}
		if tp == websocket.TextMessage {
			err := json.Unmarshal(data, &object)
			if err != nil {
				log.Error("websocket", "Error reading web socket: %v", err)
				return object, err
			}
			return object, nil
		}
	}
}

func newRequestWithHeaderSecretAndBody(url string, body any, method string, key string, value string) (*http.Request, error) {

	// Create a new request
	request, err := newRequestWithBody(url, body, method)
	if err != nil {
		log.Error("Http", "Error creating request %s: %v", url, err)
		return nil, err
	}

	request.Header.Set(key, value)
	return request, nil
}

func newRequestWithHeaderSecret(url string, method string, key string, value string) (*http.Request, error) {

	// Create a new request
	request, err := newRequest(url, method)
	if err != nil {
		log.Error("Http", "Error creating request %s: %v", url, err)
		return nil, err
	}

	request.Header.Set(key, value)
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
	return newRequestWithBody(url, body, "POST")
}

// NewPutRequest creates a new request for the given url and the body as payload. The url have to provide free access without any
// authentication. For authentication use other functions like NewPutRequestWithBearer.
func NewPutRequest(url string, body any) (*http.Request, error) {
	return newRequestWithBody(url, body, "PUT")
}

func newRequestWithBody(url string, body any, method string) (*http.Request, error) {

	// Create payload if used
	payload, err := json.Marshal(body)
	if err != nil {
		log.Error("Kafka", "Failed to marshal body: %s", err.Error())
		return nil, err
	}

	// Create a new request with payload if used
	request, err := http.NewRequest(method, url, bytes.NewBuffer(payload))
	if err != nil {
		log.Error("Http", "Error creating request %s: %v", url, err)
		return nil, err
	}

	request.Header.Set("Content-Type", "application/json")
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
	var value T

	payload, err := Do(request, timeout, checkCertificate)
	if err != nil {
		return value, err
	}

	if len(payload) == 0 {
		return value, fmt.Errorf("request returns empty payload")
	}

	err = json.Unmarshal(payload, &value)
	if err != nil {
		log.Error("Http", "Unmarshal error: %v (%s)", err, string(payload))
		return value, err
	}

	return value, nil
}

// Do return the payload returned from the request
func Do(request *http.Request, timeout time.Duration, checkCertificate bool) ([]byte, error) {

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
		return nil, err
	}

	defer func(Body io.ReadCloser) {
		err := Body.Close()
		if err != nil {
			log.Error("Http", "Error closing request for %s: %v", request.URL, err)
		}
	}(response.Body)

	// read the complete payload
	body, err := ioutil.ReadAll(response.Body)
	if err != nil {
		log.Error("Http", "Error body from %s: %v", request.URL, err)
		return nil, err
	}

	// returns the payload as string, if the status code is OK
	if response.StatusCode == http.StatusOK {
		return body, nil
	} else {
		log.Error("Http", "Error request code %d for request to %s.", response.StatusCode, request.URL)
		return nil, fmt.Errorf("error request code %d for request to %s", response.StatusCode, request.URL)
	}
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
