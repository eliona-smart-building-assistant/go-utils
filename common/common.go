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

package common

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"os/signal"
	"path/filepath"
	"regexp"
	"sync"
	"syscall"
	"time"
)

// Map to check if a function started with RunOnce is currently running.
var runOnceIds sync.Map

// RunOnce starts a function if this function not currently running. RunOnce knows, with function is currently
// running (identified by id) and skips starting the function again.
func RunOnce(function func(), id any) {
	RunOnceWithParam(func(any) {
		function()
	}, nil, id)
}

// RunOnceWithParam starts a function if this function not currently running. RunOnce knows, with function is currently
// running (identified by id) and skips starting the function again.
func RunOnceWithParam[T any](function func(T), param T, id any) {
	go func() {
		_, alreadyRuns := runOnceIds.Load(id)
		if !alreadyRuns {
			runOnceIds.Store(id, nil)
			function(param)
			runOnceIds.Delete(id)
		}
	}()
}

// WaitFor helps to start multiple functions in parallel and waits until all functions are completed.
func WaitFor(functions ...func()) {
	var waitGroup sync.WaitGroup
	waitGroup.Add(len(functions))
	for _, function := range functions {
		function := function
		go func() {
			function()
			waitGroup.Done()
		}()
	}
	waitGroup.Wait()
}

// WaitForWithOs helps to start multiple functions in parallel and waits until all functions are completed or if system signals termination
func WaitForWithOs(functions ...func()) {

	// wait group to wait for worker functions
	var waitGroup = &sync.WaitGroup{}
	waitGroup.Add(len(functions))

	// start all functions
	for _, function := range functions {

		go func(f func(), wg *sync.WaitGroup) {

			// channel to get os signals
			osSignals := make(chan os.Signal, 1)
			signal.Notify(osSignals, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)

			// channel for function
			defer wg.Done()
			functionEnds := make(chan any)

			// start function
			go func(f func(), fe chan any) {
				f()
				fe <- nil
			}(f, functionEnds)

			// wait until function ends or od signals
			select {
			case <-functionEnds:
				return
			case <-osSignals:
				return
			}

		}(function, waitGroup)

	}

	// wait until all functions ends or terminated by os
	waitGroup.Wait()
}

// Loop wraps a function in an endless loop and calls the function in the defined interval.
func Loop(function func(), interval time.Duration) func() {
	return StoppableLoop(func() bool {
		function()
		return true
	}, interval)
}

func LoopWithParam[T any](function func(T), param T, interval time.Duration) func() {
	return StoppableLoopWithParam(func(param T) bool {
		function(param)
		return true
	}, param, interval)
}

// StoppableLoop wraps a function in a loop and calls the function in the defined interval until the function return false.
func StoppableLoop(function func() bool, interval time.Duration) func() {
	return StoppableLoopWithParam(func(any) bool {
		return function()
	}, nil, interval)
}

func StoppableLoopWithParam[T any](function func(T) bool, param T, interval time.Duration) func() {
	return func() {
		osSignals := make(chan os.Signal, 1)
		defer close(osSignals)
		signal.Notify(osSignals, syscall.SIGTERM, syscall.SIGQUIT, syscall.SIGINT)
		for {
			if !function(param) {
				return
			}
			select {
			case <-time.After(interval):
			case <-osSignals:
				return
			}
		}
	}
}

// Getenv reads the value from environment variable named by key.
// If the key is not defined as environment variable the default string is returned.
func Getenv(key, fallback string) string {
	value, present := os.LookupEnv(key)
	if present {
		return value
	}
	return fallback
}

// UnmarshalFile returns the content of the file as object of type T
func UnmarshalFile[T any](path string) (T, error) {
	var object T
	data, err := ioutil.ReadFile(filepath.Join(path))
	if err != nil {
		return object, err
	}
	err = json.Unmarshal(data, &object)
	if err != nil {
		return object, err
	}
	return object, nil
}

// Ptr delivers the pointer of any constant value like Ptr("foo")
func Ptr[T any](v T) *T {
	return &v
}

// StructToMap converts a struct to map of struct properties
func StructToMap(data any) map[string]interface{} {
	d, _ := json.Marshal(&data)
	var m map[string]interface{}
	_ = json.Unmarshal(d, &m)
	return m
}

type FilterRule struct {
	Parameter string
	Regex     string
}

func filter(f func(value, rule string) (bool, error), rules [][]FilterRule, properties map[string]string) (bool, error) {
	if len(rules) == 0 {
		return true, nil
	}
	disjunction := false
	for _, disjunctionElement := range rules {
		conjunction := true
		for _, rule := range disjunctionElement {
			property, ok := properties[rule.Parameter]
			if !ok {
				// Key not present in map
				conjunction = false
				continue
			}
			match, err := f(property, rule.Regex)
			if err != nil {
				return false, fmt.Errorf("applying filter to property: %v", err)
			}
			if !match {
				conjunction = false
				continue
			}
		}
		if conjunction == true {
			disjunction = true
		}
	}
	return disjunction, nil
}

func evaluateRule(value, rule string) (bool, error) {
	r, err := regexp.Compile(rule)
	if err != nil {
		return false, fmt.Errorf("compiling rule regexp %v: %v", rule, err)
	}
	return r.MatchString(value), nil
}

// Filter checks whether the properties adhere to the rules provided.
//
// The rules are a two-dimensional array where the first dimension is joined by
// a logical disjunction ("OR") and the second one by a logical conjunction ("AND").
//
// The properties can be the for example the name and MAC address of a device.
func Filter(rules [][]FilterRule, properties map[string]string) (bool, error) {
	return filter(evaluateRule, rules, properties)
}
