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
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestStoppableLoopWithParam(t *testing.T) {
	var counter int64
	WaitFor(StoppableLoopWithParam(doUntilList10, &counter, 10))
	assert.Equal(t, int64(10), counter)
}

func TestStoppableLoop(t *testing.T) {
	var counter int64
	WaitFor(StoppableLoop(func() bool {
		return doUntilList10(&counter)
	}, 10))
	assert.Equal(t, int64(10), counter)
}

func doUntilList10(counter *int64) bool {
	atomic.AddInt64(counter, 1)
	time.Sleep(time.Millisecond * 100)
	return *counter < 10
}

func TestGetenv(t *testing.T) {
	assert.Equal(t, "default", Getenv("FOO", "default"))
	t.Setenv("FOO", "bar")
	assert.Equal(t, "bar", Getenv("FOO", "default"))
	t.Setenv("FOO", "")
	assert.Equal(t, "", Getenv("FOO", "default"))
}

func TestRunOnce(t *testing.T) {
	var counter int64 = 1
	runsOnlyOnceAtSameTime(&counter)
	assert.Equal(t, int64(2), counter)
	RunOnce(func() { runsOnlyOnceAtSameTime(&counter) }, 4711)
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, int64(3), counter)
	RunOnce(func() { runsOnlyOnceAtSameTime(&counter) }, 4711)
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, int64(3), counter)
	RunOnce(func() { runsOnlyOnceAtSameTime(&counter) }, 4712)
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, int64(4), counter)
	RunOnce(func() { runsOnlyOnceAtSameTime(&counter) }, 4711)
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, int64(4), counter)
	time.Sleep(time.Millisecond * 100)
	RunOnce(func() { runsOnlyOnceAtSameTime(&counter) }, 4711)
	time.Sleep(time.Millisecond * 10)
	assert.Equal(t, int64(5), counter)
}

func runsOnlyOnceAtSameTime(counter *int64) {
	atomic.AddInt64(counter, 1)
	time.Sleep(time.Millisecond * 100)
}

func TestFilter(t *testing.T) {
	properties := map[string]string{"x": "...", "y": "...", "z": "..."}

	testCases := []struct {
		name       string
		rules      [][]FilterRule
		properties map[string]string
		expected   bool
	}{
		{
			name:       "Empty rules",
			rules:      [][]FilterRule{},
			properties: properties,
			expected:   true,
		},
		{
			name:       "No properties match",
			rules:      [][]FilterRule{{{"x", "false"}}},
			properties: map[string]string{"anotherproperty": "..."},
			expected:   false,
		},
		{
			name:       "No rules, no properties",
			rules:      [][]FilterRule{},
			properties: map[string]string{},
			expected:   true,
		},
		{
			name: "No matches",
			rules: [][]FilterRule{
				{{"x", "false"}},
				{{"y", "false"}},
			},
			properties: properties,
			expected:   false,
		},
		{
			name: "Multiple matches",
			rules: [][]FilterRule{
				{{"x", "true"}},
				{{"y", "true"}},
			},
			properties: properties,
			expected:   true,
		},
		{
			name: "Mixed matches",
			rules: [][]FilterRule{
				{{"x", "true"}},
				{{"y", "false"}},
			},
			properties: properties,
			expected:   true,
		},
		{
			name: "Multiple conjunctions",
			rules: [][]FilterRule{
				{{"x", "true"}, {"y", "false"}},
			},
			properties: properties,
			expected:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result, err := filter(evaluateTestRule, tc.rules, tc.properties)
			if err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if result != tc.expected {
				t.Errorf("%s test failed: expected %v, got %v", tc.name, tc.expected, result)
			}
		})
	}
}

func evaluateTestRule(value, rule string) (bool, error) {
	switch rule {
	case "true":
		return true, nil
	case "false":
		return false, nil
	}
	panic("unknown rule")
}
