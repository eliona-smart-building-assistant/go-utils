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
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestValuesEqual(t *testing.T) {
	var json1 = `{"name": "John", "address": { "city": "New York", "street": "Fifth Avenue" }, "attributes": [ { "married": false }, 30 ] }`

	var json2 = `{"name": "John", "address": { "city": "New York", "street": "Fifth Avenue" }, "attributes": [ { "married": false }, 30 ] }`
	assert.Nil(t, compareDataJson([]byte(json2), []byte(json1)), "Same content and same order")
	assert.Nil(t, compareDataJson([]byte(json1), []byte(json2)), "Same content and same order")

	json2 = `{"attributes": [ { "married": false }, 30 ], "address": { "street": "Fifth Avenue", "city": "New York" }, "name": "John"}`
	assert.Nil(t, compareDataJson([]byte(json2), []byte(json1)), "Same content and different order")
	assert.Nil(t, compareDataJson([]byte(json1), []byte(json2)), "Same content and different order")

	JsonDataEquals(t, []byte(json2), []byte(json1), "Same content and different order")
	JsonDataEquals(t, []byte(json1), []byte(json2), "Same content and different order")

	json2 = `{"attributes": [ { "married": false }, 30 ], "address": { "city": "New York" }, "name": "John"}`
	assert.Nil(t, compareDataJson([]byte(json2), []byte(json1)), "Less content and different order")
	assert.Equal(t, "Missing key: street", *compareDataJson([]byte(json1), []byte(json2)), "More content and different order")
}

func TestValuesNotEqual(t *testing.T) {
	var json1 = `{"name": "John", "address": { "city": "New York", "street": "Fifth Avenue" }, "attributes": [ { "married": false }, 30 ] }`

	var json2 = `{"name": "John", "address": { "city": "New York", "street": "Sixth Avenue" }, "attributes": [ { "married": false }, 30 ] }`
	assert.Equal(t, "'address' > 'street' > Not equal: \nexpected: Sixth Avenue\nactual  : Fifth Avenue", *compareDataJson([]byte(json2), []byte(json1)))

	json2 = `{"name": "John", "address": { "city": "New York", "street": "Fifth Avenue" }, "attributes": [ { "married": false }, 31 ] }`
	assert.Equal(t, "'attributes' > Not equal: \nexpected: 31\nactual  : 30", *compareDataJson([]byte(json2), []byte(json1)))

	json2 = `{"name": "John", "address": { "city": "New York", "street": "Fifth Avenue" }, "attributes": [ { "married": true }, 30 ] }`
	assert.Equal(t, "'attributes' > 'married' > Not equal: \nexpected: true\nactual  : false", *compareDataJson([]byte(json2), []byte(json1)))
}

func TestValuesMatch(t *testing.T) {
	var json1 = `{"name": "John", "address": { "city": "New York", "street": "Fifth Avenue" }, "attributes": [ { "married": false }, 30 ] }`

	var json2 = `{"name": "J?hn", "address": { "city": "New York", "street": "Fifth .*" }, "attributes": [ { "married": false }, "[0-9]{2}" ] }`
	assert.Nil(t, compareDataJson([]byte(json2), []byte(json1)))
}

func TestValuesNotMatch(t *testing.T) {
	var json1 = `{"name": "John", "address": { "city": "New York", "street": "Fifth Avenue" }, "attributes": [ { "married": false }, 30 ] }`

	var json2 = `{"name": "J.hn", "address": { "city": "New York", "street": "Fifth .*" }, "attributes": [ { "married": false }, "[0-9]{3}" ] }`
	assert.Equal(t, "'attributes' > No match: \nexpected: [0-9]{3}\nactual  : 30", *compareDataJson([]byte(json2), []byte(json1)))

	json2 = `{"name": "J..hn", "address": { "city": "New York", "street": "Fifth .*" }, "attributes": [ { "married": false }, "[0-9]{2}" ] }`
	assert.Equal(t, "'name' > No match: \nexpected: J..hn\nactual  : John", *compareDataJson([]byte(json2), []byte(json1)))
}

func TestArrayEquals(t *testing.T) {
	var array1 = []any{map[string]any{"name": "John"}, map[string]any{"street": "Fifth Avenue"}}
	var array2 = []any{map[string]any{"name": "John"}, map[string]any{"street": "Fifth Avenue"}}
	assert.Nil(t, compareArray(array2, array1))

	array1 = []any{map[string]any{"name": "John"}, map[string]any{"street": "Fifth Avenue"}}
	array2 = []any{map[string]any{"name": "Peter"}, map[string]any{"street": "Fifth Avenue"}}
	assert.Equal(t, "'name' > Not equal: \nexpected: Peter\nactual  : John", *compareArray(array2, array1))
}
