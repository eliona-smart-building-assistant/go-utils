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

package assert

import (
	"encoding/json"
	"fmt"
	"github.com/eliona-smart-building-assistant/go-utils/common"
	"github.com/stretchr/testify/assert"
	"reflect"
	"regexp"
	"sort"
)

func JsonDataEquals(t assert.TestingT, expected, actual []byte, msgAndArgs ...any) bool {
	msg := compareDataJson(expected, actual)
	if msg != nil {
		return assert.Fail(t, *msg, msgAndArgs)
	}
	return true
}

func JsonEquals(t assert.TestingT, expected, actual map[string]any, msgAndArgs ...any) bool {
	msg := compareJson(expected, actual)
	if msg != nil {
		return assert.Fail(t, *msg, msgAndArgs)
	}
	return true
}

func ArrayEquals(t assert.TestingT, expected, actual []any, msgAndArgs ...any) bool {
	msg := compareArray(expected, actual)
	if msg != nil {
		return assert.Fail(t, *msg, msgAndArgs)
	}
	return true
}

func ArrayEqualsStrictOrder(t assert.TestingT, expected, actual []any, msgAndArgs ...any) bool {
	msg := compareArrayStrictOrder(expected, actual)
	if msg != nil {
		return assert.Fail(t, *msg, msgAndArgs)
	}
	return true
}

func compareDataJson(expected, actual []byte) *string {
	var objActual, objExpected map[string]any
	if err := json.Unmarshal(actual, &objActual); err != nil {
		return common.Ptr(err.Error())
	}
	if err := json.Unmarshal(expected, &objExpected); err != nil {
		return common.Ptr(err.Error())
	}
	return compareJson(objExpected, objActual)
}

func compareJson(expected, actual map[string]any) *string {
	for keyExpected, valueExpected := range expected {
		if valueActual, ok := actual[keyExpected]; ok {
			msg := compareAny(valueExpected, valueActual)
			if msg != nil {
				msg2 := fmt.Sprintf("'%s' > %s", keyExpected, *msg)
				return &msg2
			}
		} else {
			msg := fmt.Sprintf("Missing key '%s'", keyExpected)
			return &msg
		}
	}
	return nil
}

type ByString []any

func (a ByString) Len() int { return len(a) }

func (a ByString) Swap(i, j int) { a[i], a[j] = a[j], a[i] }

func (a ByString) Less(i, j int) bool {
	x := fmt.Sprintf("%v", a[i])
	y := fmt.Sprintf("%v", a[j])
	return x < y
}

func compareArray(expected, actual []any) *string {
	sort.Sort(ByString(expected))
	sort.Sort(ByString(actual))
	return compareArrayStrictOrder(expected, actual)
}

func compareArrayStrictOrder(expected, actual []any) *string {
	if len(actual) != len(expected) {
		msg := fmt.Sprintf("Length not equal: \n"+
			"expected: %v\n"+
			"actual  : %v", len(expected), len(actual))
		return &msg
	}
	for i := range expected {
		msg := compareAny(expected[i], actual[i])
		if msg != nil {
			return msg
		}
	}
	return nil
}

func compareAny(expected, actual any) *string {
	switch valueExpected := expected.(type) {
	case map[string]any:
		if valueActual, ok := actual.(map[string]any); ok {
			msg := compareJson(valueExpected, valueActual)
			if msg != nil {
				return msg
			}
		} else {
			msg := fmt.Sprintf("Wrong type: \n"+
				"expected: map[string]interface{}\n"+
				"actual  : %v", reflect.TypeOf(expected))
			return &msg
		}
	case []any:
		if valueActual, ok := actual.([]any); ok {
			msg := compareArray(valueExpected, valueActual)
			if msg != nil {
				return msg
			}
		} else {
			msg := fmt.Sprintf("Wrong type: \n"+
				"expected: []interface{}\n"+
				"actual  : %v", reflect.TypeOf(expected))
			return &msg
		}
	case string:
		msg := compareValue(valueExpected, actual)
		if msg != nil {
			return msg
		}
	default:
		if !reflect.DeepEqual(actual, expected) {
			msg := fmt.Sprintf("Not equal: \n"+
				"expected: %v\n"+
				"actual  : %v", expected, actual)
			return &msg
		}
	}
	return nil
}

func compareValue(expected string, actual any) *string {
	if regexp.QuoteMeta(expected) == expected {
		if expected != fmt.Sprintf("%v", actual) {
			msg := fmt.Sprintf("Not equal: \n"+
				"expected: %v\n"+
				"actual  : %s", expected, actual)
			return &msg
		}
		return nil
	}
	re := regexp.MustCompile(expected)
	if !re.MatchString(fmt.Sprintf("%v", actual)) {
		msg := fmt.Sprintf("No match: \n"+
			"expected: %s\n"+
			"actual  : %v", expected, actual)
		return &msg
	}
	return nil
}
