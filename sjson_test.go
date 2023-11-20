package sjson

import (
	"encoding/hex"
	"errors"
	"fmt"
	"math/rand"
	"testing"
	"time"

	"github.com/tidwall/gjson"
	"github.com/tidwall/pretty"
)

const (
	setRaw    = 1
	setBool   = 2
	setInt    = 3
	setFloat  = 4
	setString = 5
	setDelete = 6
)

func sortJSON(json string) string {
	opts := pretty.Options{SortKeys: true}
	return string(pretty.Ugly(pretty.PrettyOptions([]byte(json), &opts)))
}

func testRaw(t *testing.T, kind int, expect, json, path string, value interface{}) {
	t.Helper()
	expect = sortJSON(expect)
	var json2 string
	var err error
	switch kind {
	default:
		json2, err = Set(json, path, value)
	case setRaw:
		json2, err = SetRaw(json, path, value.(string))
	case setDelete:
		json2, err = Delete(json, path)
	}

	if err != nil {
		t.Fatal(err)
	}
	json2 = sortJSON(json2)
	if json2 != expect {
		t.Fatalf("expected '%v', got '%v'", expect, json2)
	}
	var json3 []byte
	switch kind {
	default:
		json3, err = SetBytes([]byte(json), path, value)
	case setRaw:
		json3, err = SetRawBytes([]byte(json), path, []byte(value.(string)))
	case setDelete:
		json3, err = DeleteBytes([]byte(json), path)
	}
	json3 = []byte(sortJSON(string(json3)))
	if err != nil {
		t.Fatal(err)
	} else if string(json3) != expect {
		t.Fatalf("expected '%v', got '%v'", expect, string(json3))
	}
}
func TestBasic(t *testing.T) {
	testRaw(t, setRaw, `[{"hiw":"planet","hi":"world"}]`, `[{"hi":"world"}]`, "0.hiw", `"planet"`)
	testRaw(t, setRaw, `[true]`, ``, "0", `true`)
	testRaw(t, setRaw, `[null,true]`, ``, "1", `true`)
	testRaw(t, setRaw, `[1,null,true]`, `[1]`, "2", `true`)
	testRaw(t, setRaw, `[1,true,false]`, `[1,null,false]`, "1", `true`)
	testRaw(t, setRaw,
		`[1,{"hello":"when","this":[0,null,2]},false]`,
		`[1,{"hello":"when","this":[0,1,2]},false]`,
		"1.this.1", `null`)
	testRaw(t, setRaw,
		`{"a":1,"b":{"hello":"when","this":[0,null,2]},"c":false}`,
		`{"a":1,"b":{"hello":"when","this":[0,1,2]},"c":false}`,
		"b.this.1", `null`)
	testRaw(t, setRaw,
		`{"a":1,"b":{"hello":"when","this":[0,null,2,null,4]},"c":false}`,
		`{"a":1,"b":{"hello":"when","this":[0,null,2]},"c":false}`,
		"b.this.4", `4`)
	testRaw(t, setRaw,
		`{"b":{"this":[null,null,null,null,4]}}`,
		``,
		"b.this.4", `4`)
	testRaw(t, setRaw,
		`[null,{"this":[null,null,null,null,4]}]`,
		``,
		"1.this.4", `4`)
	testRaw(t, setRaw,
		`{"1":{"this":[null,null,null,null,4]}}`,
		``,
		":1.this.4", `4`)
	testRaw(t, setRaw,
		`{":1":{"this":[null,null,null,null,4]}}`,
		``,
		"\\:1.this.4", `4`)
	testRaw(t, setRaw,
		`{":\\1":{"this":[null,null,null,null,{".HI":4}]}}`,
		``,
		"\\:\\\\1.this.4.\\.HI", `4`)
	testRaw(t, setRaw,
		`{"app.token":"cde"}`,
		`{"app.token":"abc"}`,
		"app\\.token", `"cde"`)
	testRaw(t, setRaw,
		`{"b":{"this":{"😇":""}}}`,
		``,
		"b.this.😇", `""`)
	testRaw(t, setRaw,
		`[ 1,2  ,3]`,
		`  [ 1,2  ] `,
		"-1", `3`)
	testRaw(t, setInt, `[1234]`, ``, `0`, int64(1234))
	testRaw(t, setFloat, `[1234.5]`, ``, `0`, float64(1234.5))
	testRaw(t, setString, `["1234.5"]`, ``, `0`, "1234.5")
	testRaw(t, setBool, `[true]`, ``, `0`, true)
	testRaw(t, setBool, `[null]`, ``, `0`, nil)
	testRaw(t, setString, `{"arr":[1]}`, ``, `arr.-1`, 1)
	testRaw(t, setString, `{"a":"\\"}`, ``, `a`, "\\")
	testRaw(t, setString, `{"a":"C:\\Windows\\System32"}`, ``, `a`, `C:\Windows\System32`)
}

func TestDelete(t *testing.T) {
	testRaw(t, setDelete, `[456]`, `[123,456]`, `0`, nil)
	testRaw(t, setDelete, `[123,789]`, `[123,456,789]`, `1`, nil)
	testRaw(t, setDelete, `[123,456]`, `[123,456,789]`, `-1`, nil)
	testRaw(t, setDelete, `{"a":[123,456]}`, `{"a":[123,456,789]}`, `a.-1`, nil)
	testRaw(t, setDelete, `{"and":"another"}`, `{"this":"that","and":"another"}`, `this`, nil)
	testRaw(t, setDelete, `{"this":"that"}`, `{"this":"that","and":"another"}`, `and`, nil)
	testRaw(t, setDelete, `{}`, `{"and":"another"}`, `and`, nil)
	testRaw(t, setDelete, `{"1":"2"}`, `{"1":"2"}`, `3`, nil)
}

// TestRandomData is a fuzzing test that throws random data at SetRaw
// function looking for panics.
func TestRandomData(t *testing.T) {
	var lstr string
	defer func() {
		if v := recover(); v != nil {
			println("'" + hex.EncodeToString([]byte(lstr)) + "'")
			println("'" + lstr + "'")
			panic(v)
		}
	}()
	rand.Seed(time.Now().UnixNano())
	b := make([]byte, 200)
	for i := 0; i < 2000000; i++ {
		n, err := rand.Read(b[:rand.Int()%len(b)])
		if err != nil {
			t.Fatal(err)
		}
		lstr = string(b[:n])
		SetRaw(lstr, "zzzz.zzzz.zzzz", "123")
	}
}

func TestDeleteIssue21(t *testing.T) {
	json := `{"country_code_from":"NZ","country_code_to":"SA","date_created":"2018-09-13T02:56:11.25783Z","date_updated":"2018-09-14T03:15:16.67356Z","disabled":false,"last_edited_by":"Developers","id":"a3e...bc454","merchant_id":"f2b...b91abf","signed_date":"2018-02-01T00:00:00Z","start_date":"2018-03-01T00:00:00Z","url":"https://www.google.com"}`
	res1 := gjson.Get(json, "date_updated")
	var err error
	json, err = Delete(json, "date_updated")
	if err != nil {
		t.Fatal(err)
	}
	res2 := gjson.Get(json, "date_updated")
	res3 := gjson.Get(json, "date_created")
	if !res1.Exists() || res2.Exists() || !res3.Exists() {
		t.Fatal("bad news")
	}

	// We change the number of characters in this to make the section of the string before the section that we want to delete a certain length

	// ---------------------------
	lenBeforeToDeleteIs307AsBytes := `{"1":"","0":"012345678901234567890123456789012345678901234567890123456789012345678901234567","to_delete":"0","2":""}`

	expectedForLenBefore307AsBytes := `{"1":"","0":"012345678901234567890123456789012345678901234567890123456789012345678901234567","2":""}`
	// ---------------------------

	// ---------------------------
	lenBeforeToDeleteIs308AsBytes := `{"1":"","0":"0123456789012345678901234567890123456789012345678901234567890123456789012345678","to_delete":"0","2":""}`

	expectedForLenBefore308AsBytes := `{"1":"","0":"0123456789012345678901234567890123456789012345678901234567890123456789012345678","2":""}`
	// ---------------------------

	// ---------------------------
	lenBeforeToDeleteIs309AsBytes := `{"1":"","0":"01234567890123456789012345678901234567890123456789012345678901234567890123456","to_delete":"0","2":""}`

	expectedForLenBefore309AsBytes := `{"1":"","0":"01234567890123456789012345678901234567890123456789012345678901234567890123456","2":""}`
	// ---------------------------

	var data = []struct {
		desc     string
		input    string
		expected string
	}{
		{
			desc:     "len before \"to_delete\"... = 307",
			input:    lenBeforeToDeleteIs307AsBytes,
			expected: expectedForLenBefore307AsBytes,
		},
		{
			desc:     "len before \"to_delete\"... = 308",
			input:    lenBeforeToDeleteIs308AsBytes,
			expected: expectedForLenBefore308AsBytes,
		},
		{
			desc:     "len before \"to_delete\"... = 309",
			input:    lenBeforeToDeleteIs309AsBytes,
			expected: expectedForLenBefore309AsBytes,
		},
	}

	for i, d := range data {
		result, err := Delete(d.input, "to_delete")

		if err != nil {
			t.Error(fmtErrorf(testError{
				unexpected: "error",
				desc:       d.desc,
				i:          i,
				lenInput:   len(d.input),
				input:      d.input,
				expected:   d.expected,
				result:     result,
			}))
		}
		if result != d.expected {
			t.Error(fmtErrorf(testError{
				unexpected: "result",
				desc:       d.desc,
				i:          i,
				lenInput:   len(d.input),
				input:      d.input,
				expected:   d.expected,
				result:     result,
			}))
		}
	}
}

type testError struct {
	unexpected string
	desc       string
	i          int
	lenInput   int
	input      interface{}
	expected   interface{}
	result     interface{}
}

func fmtErrorf(e testError) string {
	return fmt.Sprintf(
		"Unexpected %s:\n\t"+
			"for=%q\n\t"+
			"i=%d\n\t"+
			"len(input)=%d\n\t"+
			"input=%v\n\t"+
			"expected=%v\n\t"+
			"result=%v",
		e.unexpected, e.desc, e.i, e.lenInput, e.input, e.expected, e.result,
	)
}

func TestSetDotKeyIssue10(t *testing.T) {
	json := `{"app.token":"abc"}`
	json, _ = Set(json, `app\.token`, "cde")
	if json != `{"app.token":"cde"}` {
		t.Fatalf("expected '%v', got '%v'", `{"app.token":"cde"}`, json)
	}
}
func TestDeleteDotKeyIssue19(t *testing.T) {
	json := []byte(`{"data":{"key1":"value1","key2.something":"value2"}}`)
	json, _ = DeleteBytes(json, `data.key2\.something`)
	if string(json) != `{"data":{"key1":"value1"}}` {
		t.Fatalf("expected '%v', got '%v'", `{"data":{"key1":"value1"}}`, json)
	}
}

func TestIssue36(t *testing.T) {
	var json = `
	{
	    "size": 1000
    }
`
	var raw = `
	{
	    "sample": "hello"
	}
`
	_ = raw
	if true {
		json, _ = SetRaw(json, "aggs", raw)
	}
	if !gjson.Valid(json) {
		t.Fatal("invalid json")
	}
	res := gjson.Get(json, "aggs.sample").String()
	if res != "hello" {
		t.Fatal("unexpected result")
	}
}

var example = `
{
	"name": {"first": "Tom", "last": "Anderson"},
	"age":37,
	"children": ["Sara","Alex","Jack"],
	"fav.movie": "Deer Hunter",
	"friends": [
	  {"first": "Dale", "last": "Murphy", "age": 44, "nets": ["ig", "fb", "tw"]},
	  {"first": "Roger", "last": "Craig", "age": 68, "nets": ["fb", "tw"]},
	  {"first": "Jane", "last": "Murphy", "age": 47, "nets": ["ig", "tw"]}
	]
  }
  `

func TestIndex(t *testing.T) {
	path := `friends.#(last="Murphy").last`
	json, err := Set(example, path, "Johnson")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.Get(json, "friends.#.last").String() != `["Johnson","Craig","Murphy"]` {
		t.Fatal("mismatch")
	}
}

func TestIndexes(t *testing.T) {
	path := `friends.#(last="Murphy")#.last`
	json, err := Set(example, path, "Johnson")
	if err != nil {
		t.Fatal(err)
	}
	if gjson.Get(json, "friends.#.last").String() != `["Johnson","Craig","Johnson"]` {
		t.Fatal("mismatch")
	}
}

func TestIssue61(t *testing.T) {
	json := `{
		"@context": {
		  "rdfs": "http://www.w3.org/2000/01/rdf-schema#",
		  "@vocab": "http://schema.org/",
		  "sh": "http://www.w3.org/ns/shacl#"
		}
	}`
	json1, _ := Set(json, "@context.@vocab", "newval")
	if gjson.Get(json1, "@context.@vocab").String() != "newval" {
		t.Fail()
	}
}

func TestSetBytesOptionsManyByGetResult(t *testing.T) {
	tcs := []struct {
		name     string
		jsonPath string
		newIDs   []interface{}
		actual   string
		expected string
	}{
		{
			name:     "string",
			jsonPath: "#.id",
			newIDs:   []interface{}{"stringid1", "stringid2", "stringid3"},
			actual: `[
	  {"id": "id1","first": "Dale", "last": "Murphy", "age": 44, "nets": ["ig", "fb", "tw"]},
	  {"id": "id2","first": "Roger", "last": "Craig", "age": 68, "nets": ["fb", "tw"]},
	  {"id": "id3","first": "Jane", "last": "Murphy", "age": 47, "nets": ["ig", "tw"]}
	]`,
			expected: `[
	  {"id": "stringid1","first": "Dale", "last": "Murphy", "age": 44, "nets": ["ig", "fb", "tw"]},
	  {"id": "stringid2","first": "Roger", "last": "Craig", "age": 68, "nets": ["fb", "tw"]},
	  {"id": "stringid3","first": "Jane", "last": "Murphy", "age": 47, "nets": ["ig", "tw"]}
	]`,
		},
		{
			name:     "bool",
			jsonPath: "#.isAdult",
			newIDs:   []interface{}{false, false, false},
			actual: `[
	  {"id": "id1","first": "Dale", "last": "Murphy", "age": 44, "isAdult": true, "nets": ["ig", "fb", "tw"]},
	  {"id": "id2","first": "Roger", "last": "Craig", "age": 68, "isAdult": true, "nets": ["fb", "tw"]},
	  {"id": "id3","first": "Jane", "last": "Murphy", "age": 47, "isAdult": true, "nets": ["ig", "tw"]}
	]`,
			expected: `[
	  {"id": "id1","first": "Dale", "last": "Murphy", "age": 44, "isAdult": false, "nets": ["ig", "fb", "tw"]},
	  {"id": "id2","first": "Roger", "last": "Craig", "age": 68, "isAdult": false, "nets": ["fb", "tw"]},
	  {"id": "id3","first": "Jane", "last": "Murphy", "age": 47, "isAdult": false, "nets": ["ig", "tw"]}
	]`,
		},
		{
			name:     "int",
			jsonPath: "#.age",
			newIDs:   []interface{}{10, 20, 30},
			actual: `[
	  {"id": "id1","first": "Dale", "last": "Murphy", "age": 44, "nets": ["ig", "fb", "tw"]},
	  {"id": "id2","first": "Roger", "last": "Craig", "age": 68, "nets": ["fb", "tw"]},
	  {"id": "id3","first": "Jane", "last": "Murphy", "age": 47, "nets": ["ig", "tw"]}
	]`,
			expected: `[
	  {"id": "id1","first": "Dale", "last": "Murphy", "age": 10, "nets": ["ig", "fb", "tw"]},
	  {"id": "id2","first": "Roger", "last": "Craig", "age": 20, "nets": ["fb", "tw"]},
	  {"id": "id3","first": "Jane", "last": "Murphy", "age": 30, "nets": ["ig", "tw"]}
	]`,
		},
		{
			name:     "float",
			jsonPath: "#.age",
			newIDs:   []interface{}{10.1, 20.1, 30.1},
			actual: `[
	  {"id": "id1","first": "Dale", "last": "Murphy", "age": 44.1, "nets": ["ig", "fb", "tw"]},
	  {"id": "id2","first": "Roger", "last": "Craig", "age": 68.1, "nets": ["fb", "tw"]},
	  {"id": "id3","first": "Jane", "last": "Murphy", "age": 471., "nets": ["ig", "tw"]}
	]`,
			expected: `[
	  {"id": "id1","first": "Dale", "last": "Murphy", "age": 10.1, "nets": ["ig", "fb", "tw"]},
	  {"id": "id2","first": "Roger", "last": "Craig", "age": 20.1, "nets": ["fb", "tw"]},
	  {"id": "id3","first": "Jane", "last": "Murphy", "age": 30.1, "nets": ["ig", "tw"]}
	]`,
		},
	}

	for _, tc := range tcs {
		t.Run(tc.name, func(t *testing.T) {
			getResult := gjson.Get(tc.actual, tc.jsonPath)
			if !getResult.Exists() || !getResult.IsArray() {
				t.Fail()
			}
			arrayResult := getResult.Array()

			opts := &Options{
				Optimistic:     true,
				ReplaceInPlace: true,
			}
			actual, err := SetBytesOptionsManyByGetResult([]byte(tc.actual), arrayResult, tc.newIDs, opts)
			if err != nil {
				t.Fatal(err)
			}

			if sortJSON(tc.expected) != sortJSON(string(actual)) {
				t.Fatalf("expected '%v', got '%v'", tc.expected, string(actual))
			}
		})
	}
}

func TestDeleteMany(t *testing.T) {
	tests := []struct {
		name      string
		body      string
		jsonPaths []string
		resExp    string
		errExp    error
	}{
		{
			name:      "success_nested_object_value_simple",
			body:      `{"object":{"nestedObject":{"name":"test","value":15,"deleted":true}}}`,
			jsonPaths: []string{"object.nestedObject.value", "object.nestedObject.name"},
			resExp:    `{"object":{"nestedObject":{"deleted":true}}}`,
			errExp:    nil,
		},
		{
			name:      "success_nested_object_value_array",
			body:      `{"object":{"nestedObject":{"name":"test","value":[{"name":"array1","value":1},{"name":"array2","value":2}]}}}`,
			jsonPaths: []string{"object.nestedObject.value"},
			resExp:    `{"object":{"nestedObject":{"name":"test"}}}`,
			errExp:    nil,
		},
		{
			name:      "success_array_nested_object_field",
			body:      `[{"name":"object1","value":1,"nested":{"id":"one","desc":"nested one"}},{"name":"object2","value":19,"nested":{"id":"two","desc":"nested two"}}]`,
			jsonPaths: []string{"#.nested.desc", "#.nonexistent"},
			resExp:    `[{"name":"object1","value":1,"nested":{"id":"one"}},{"name":"object2","value":19,"nested":{"id":"two"}}]`,
			errExp:    nil,
		},
		{
			name:      "success_array_nested_object_field_array",
			body:      `[{"name":"object1","value":1,"nested":{"id":"one","value":[{"name":"array1","value":1},{"name":"array2","value":2}]}},{"name":"object2","value":19,"nested":{"id":"two","desc":"nested two","value":[{"name":"array1","value":1}]}}]`,
			jsonPaths: []string{"#.nested.value", "#.nonexistent"},
			resExp:    `[{"name":"object1","value":1,"nested":{"id":"one"}},{"name":"object2","value":19,"nested":{"id":"two","desc":"nested two"}}]`,
			errExp:    nil,
		},
		{
			name:      "success_object_nested_array_field",
			body:      `{"name":"object1","value":1,"nestedArray":[{"id":"one","desc":"nested one","value":15},{"id":"two","desc":"nested two","value":55}]}`,
			jsonPaths: []string{"nestedArray.#.value", "nestedArray.#.desc"},
			resExp:    `{"name":"object1","value":1,"nestedArray":[{"id":"one"},{"id":"two"}]}`,
			errExp:    nil,
		},
		{
			name:      "success_twice_nested_array_field",
			body:      `{"name":"object1","value":1,"nestedArray":[{"id":"one","nestedArray":[{"name":"nestedOne1","desc":"nested 1 one"},{"name":"nestedOne2"}]},{"id":"two","nestedArray":[{"name":"nestedTwo1","desc":"nested 2 one"},{"name":"nestedTwo2","desc":"nested 2 two"}]}]}`,
			jsonPaths: []string{"nestedArray.#.nestedArray.#.desc", "value"},
			resExp:    `{"name":"object1","nestedArray":[{"id":"one","nestedArray":[{"name":"nestedOne1"},{"name":"nestedOne2"}]},{"id":"two","nestedArray":[{"name":"nestedTwo1"},{"name":"nestedTwo2"}]}]}`,
			errExp:    nil,
		},
		{
			name:      "success_many_times_nested_array_field",
			body:      `[{"name":"object1","value":1,"nestedArray":[{"id":"one","nestedArray":[{"name":"nested1","desc":"nested 1","nestedArray":[{"name":"nested 2","desc":"nested 2","nestedArray":[{"name":"nested 3","desc":"nested 2"},{"name":"nested 4","desc":"nested 3"}]},{"name":"nested 5","desc":"nested 5"}]},{"name":"nested 6","desc":"nested 6","nestedArray":[{"name":"nested 7","desc":"nested 7","nestedArray":[{"name":"nested 8","desc":"nested 8"},{"name":"nested 9","desc":"nested 9"},{"name":"nested 10"}]},{"name":"nested 11","desc":"nested 11","nestedArray":[{"name":"nested 12","desc":"nested 12","nestedArray":[{"name":"nested 11111111","desc":"nested 11111"}]}]},{"name":"nested 12","desc":"nested 12"}]}]},{"id":"two","nestedArray":[{"name":"nested 13","desc":"nested 13","nestedArray":[{"name":"nested 14","desc":"nested 14"},{"name":"nested 15","desc":"nested 15"}]},{"name":"nested 16","desc":"nested 16"}]}]},{"name":"object2","value":2,"nestedArray":[{"id":"one","nestedArray":[{"name":"nested 17","desc":"nested 17","nestedArray":[{"name":"nested 18","desc":"nested 18","nestedArray":[{"name":"nested 19","desc":"nested 19"},{"name":"nested 20","desc":"nested 20"}]},{"name":"nested 21","desc":"nested 21"}]},{"name":"nested 22","desc":"nested 22","nestedArray":[{"name":"nested 23","desc":"nested 23"},{"name":"nested 24","desc":"nested 24"},{"name":"nested 25","desc":"nested 25","nestedArray":[{"name":"nested 26","desc":"nested 26"},{"name":"nested 27","desc":"nested 27"}]}]}]},{"id":"two","nestedArray":[{"name":"nested 28","desc":"nested 28","nestedArray":[{"name":"nested 29","desc":"nested 29"},{"name":"nested 30","desc":"nested 30","nestedArray":[{"name":"nested 31","desc":"nested 31","nestedArray":[{"name":"nested 32","desc":"nested 32"},{"name":"nested 33","desc":"nested 33"}]},{"name":"nested 34","desc":"nested 34"}]}]},{"name":"nested22","desc":"nested 22"}]}]}]`,
			jsonPaths: []string{"#.nestedArray.#.nestedArray.#.nestedArray.#.nestedArray.#.desc", "#.value", "#.nestedArray.#.nestedArray.#.nestedArray.#.nestedArray.#.nestedArray.#.desc"},
			resExp:    `[{"name":"object1","nestedArray":[{"id":"one","nestedArray":[{"name":"nested1","desc":"nested 1","nestedArray":[{"name":"nested 2","desc":"nested 2","nestedArray":[{"name":"nested 3"},{"name":"nested 4"}]},{"name":"nested 5","desc":"nested 5"}]},{"name":"nested 6","desc":"nested 6","nestedArray":[{"name":"nested 7","desc":"nested 7","nestedArray":[{"name":"nested 8"},{"name":"nested 9"},{"name":"nested 10"}]},{"name":"nested 11","desc":"nested 11","nestedArray":[{"name":"nested 12","nestedArray":[{"name":"nested 11111111"}]}]},{"name":"nested 12","desc":"nested 12"}]}]},{"id":"two","nestedArray":[{"name":"nested 13","desc":"nested 13","nestedArray":[{"name":"nested 14","desc":"nested 14"},{"name":"nested 15","desc":"nested 15"}]},{"name":"nested 16","desc":"nested 16"}]}]},{"name":"object2","nestedArray":[{"id":"one","nestedArray":[{"name":"nested 17","desc":"nested 17","nestedArray":[{"name":"nested 18","desc":"nested 18","nestedArray":[{"name":"nested 19"},{"name":"nested 20"}]},{"name":"nested 21","desc":"nested 21"}]},{"name":"nested 22","desc":"nested 22","nestedArray":[{"name":"nested 23","desc":"nested 23"},{"name":"nested 24","desc":"nested 24"},{"name":"nested 25","desc":"nested 25","nestedArray":[{"name":"nested 26"},{"name":"nested 27"}]}]}]},{"id":"two","nestedArray":[{"name":"nested 28","desc":"nested 28","nestedArray":[{"name":"nested 29","desc":"nested 29"},{"name":"nested 30","desc":"nested 30","nestedArray":[{"name":"nested 31","nestedArray":[{"name":"nested 32"},{"name":"nested 33"}]},{"name":"nested 34"}]}]},{"name":"nested22","desc":"nested 22"}]}]}]`,
			errExp:    nil,
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			res, err := DeleteMany(test.body, test.jsonPaths)
			if !errors.Is(err, test.errExp) {
				t.Fatalf("expected %v error, got - %v", test.errExp, err)
			}

			if res != test.resExp {
				t.Fatalf("expected %v result, got - %v", test.resExp, res)
			}
		})
	}
}
