// Copyright 2019 F5 Networks. All rights reserved.
// Use of this source code is governed by a MIT-style
// license that can be found in the LICENSE file.

package decode_test

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"testing"
	"time"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/weberr13/go-decode/decode"
)

var testSPF = func(s string) (func(map[string]interface{}) (interface{}, error), error) { return nil, nil }
var testErrSPF = func(s string) (func(map[string]interface{}) (interface{}, error), error) { return testErrFactory, nil }
var testErrFactory = func(o map[string]interface{}) (interface{}, error) { return nil, errors.New("Wrong Discriminator") }

type StructWithDefaults struct {
	Val     string     `default:"STRING_VAL"`
	Ptr     *string    `default:"STRING_PTR"`
	Int     int        `default:"-7"`
	UInt    uint       `default:"12"`
	Time    time.Time  `default:"2019-10-28T12:35:56Z"`
	PTime   *time.Time `default:"2019-10-28T23:45:10Z"`
	Int8    int8       `default:"127"`
	UInt8   uint8      `default:"127"`
	Int16   int16      `default:"32767"`
	UInt16  uint16     `default:"65535"`
	Int32   int32      `default:"2147483647"`
	UInt32  uint32     `default:"4294967295"`
	Int64   int64      `default:"32767"`
	UInt64  uint64     `default:"65535"`
	Float32 float32    `default:"1.0"`
	Float64 float64    `default:"1.0"`
	Bool    bool       `default:"true"`

	SR SubRecord
}

type SubRecord struct {
	kind string
	Name *string `default:"James"`
}

type MyString string

func NewSubRecord() interface{} {
	encapsulated := "foo"
	return &SubRecord{
		kind: "sub_record",
		Name: &encapsulated,
	}
}

type SubRecord2 struct {
	kind    string
	Name    MyString
	PtrName *MyString
	Subs    []SubRecord
}

func (r SubRecord2) Discriminator() string {
	return string(r.kind)
}

func NewSubRecord2() interface{} {
	return &SubRecord2{
		kind: "sub_record2",
	}
}

type Record struct {
	kind     string
	Name     string
	Optional *string
	Num      *int
	Slice    []string
	Sub      interface{}
}

func (r Record) Discriminator() string {
	return r.kind
}

func NewRecord() interface{} {
	return &Record{
		kind: "record",
	}
}

type Envelope struct {
	Owners []*PetOwner
}

type LivesInRequiredArray struct {
	Name    string
	LivesIn []string
}

type LivesInRequiredArrayNested struct {
	Name    string
	LivesIn []RequiredBasicTypes
}

type LivesInArrayOfPointers struct {
	Name    string
	LivesIn []*string
}

type RequiredBasicTypes struct {
	Age  int
	Name string
	Lost bool
}

type LivesInStruct struct {
	LivesIn *RequiredBasicTypes
}

type TimedStruct struct {
	Name       string
	UpdateTime *time.Time
}

func MyTestFactory(kind string) (interface{}, error) {
	fm := map[string]func() interface{}{
		"record":      NewRecord,
		"sub_record":  NewSubRecord,
		"sub_record2": NewSubRecord2,
	}
	f, ok := fm[kind]
	if !ok {
		return nil, fmt.Errorf("cannot find type %s", kind)
	}
	return f(), nil
}

func TestDecodeNestedObject(t *testing.T) {

	m := map[string]interface{}{
		"name":  "foo",
		"kind":  "record",
		"slice": []string{"foo", "bar"},
		"sub": map[string]interface{}{
			"name": "bar",
			"kind": "sub_record",
		},
	}
	Convey("wrong discriminator, doesn't exist", t, func() {
		_, err := decode.Decode(m, "kib", MyTestFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("wrong discriminator, not a type", t, func() {
		_, err := decode.Decode(m, "name", MyTestFactory)
		So(err, ShouldNotBeNil)
	})

	// todo: this panics because the expectation is that sub is an object, not base type, but should defend
	//Convey("unrully child object - assigned wrong type", t, func() {
	//	mp := map[string]interface{}{
	//		"name":  "foo",
	//		"kind":  "record",
	//		"slice": []string{"foo", "bar"},
	//		"sub": "12",
	//	}
	//	_, err := decode.Decode(mp, "kind", MyTestFactory)
	//	So(err, ShouldNotBeNil)
	//})
	Convey("unrully child object", t, func() {
		mp := map[string]interface{}{
			"name":  "foo",
			"kind":  "record",
			"slice": []string{"foo", "bar"},
			"sub": map[string]interface{}{
				"name": "bar",
				"kind": "unknown",
			},
		}
		_, err := decode.Decode(mp, "kind", MyTestFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("decode unruly child object in slice", t, func() {
		mp := map[string]interface{}{
			"name":  "foo",
			"kind":  "record",
			"slice": []string{"foo", "bar"},
			"sub": map[string]interface{}{
				"subs": []map[string]interface{}{
					{
						"kind": "unknown",
						"name": "1",
					},
				},
				"kind": "sub_record2",
			},
		}
		_, err := decode.Decode(mp, "kind", MyTestFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("unmarshal unruly child object in slice", t, func() {
		mp := map[string]interface{}{
			"name":  "foo",
			"kind":  "record",
			"slice": []string{"foo", "bar"},
			"sub": map[string]interface{}{
				"subs": []map[string]interface{}{
					{
						"kind": "unknown",
						"name": "1",
					},
				},
				"kind": "sub_record2",
			},
		}
		b, err := json.Marshal(mp)
		So(err, ShouldBeNil)
		_, err = decode.UnmarshalJSON(b, "kind", MyTestFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("Decode a nested object", t, func() {
		dec, err := decode.Decode(m, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		name := "bar"
		So(rec, ShouldResemble, &Record{
			kind:  "record",
			Name:  "foo",
			Slice: []string{"foo", "bar"},
			Sub:   &SubRecord{kind: "sub_record", Name: &name},
		})
	})
	Convey("Unmarshal a nested object, different subtype", t, func() {
		m := map[string]interface{}{
			"name":  "foo",
			"kind":  "record",
			"slice": []string{"foo", "bar"},
			"sub": map[string]interface{}{
				"subs": []map[string]interface{}{
					{
						"kind": "sub_record",
						"name": "1",
					},
				},
				"kind":     "sub_record2",
				"ptr_name": "sub_record2",
				"name":     "sub_record2",
			},
		}
		b, err := json.Marshal(m)
		So(err, ShouldBeNil)
		dec, err := decode.UnmarshalJSON(b, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		encapsulated := MyString("sub_record2")
		strName := "1"
		So(rec, ShouldResemble, &Record{
			kind:  "record",
			Name:  "foo",
			Slice: []string{"foo", "bar"},
			Sub: &SubRecord2{
				kind:    "sub_record2",
				PtrName: &encapsulated,
				Name:    encapsulated,
				Subs: []SubRecord{
					SubRecord{
						kind: "sub_record",
						Name: &strName,
					},
				},
			},
		})
	})
	Convey("Decode a nested object, different subtype", t, func() {
		m := map[string]interface{}{
			"name":  "foo",
			"kind":  "record",
			"slice": []string{"foo", "bar"},
			"sub": map[string]interface{}{
				"subs": []map[string]interface{}{
					{
						"kind": "sub_record",
						"name": "1",
					},
				},
				"kind": "sub_record2",
			},
		}
		dec, err := decode.Decode(m, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		strName := "1"
		So(rec, ShouldResemble, &Record{
			kind:  "record",
			Name:  "foo",
			Slice: []string{"foo", "bar"},
			Sub: &SubRecord2{
				kind:    "sub_record2",
				PtrName: nil,
				Name:    "",
				Subs: []SubRecord{
					SubRecord{
						kind: "sub_record",
						Name: &strName,
					},
				},
			},
		})
	})
	Convey("Decode a nested object, different subtype, pointer and aliased type values", t, func() {
		m := map[string]interface{}{
			"name":  "foo",
			"kind":  "record",
			"slice": []string{"foo", "bar"},
			"sub": map[string]interface{}{
				"subs": []map[string]interface{}{
					{
						"kind": "sub_record",
						"name": "1",
					},
				},
				"kind":     "sub_record2",
				"ptr_name": "sub_record2",
				"name":     "sub_record2",
			},
		}
		dec, err := decode.Decode(m, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		encapsulated := MyString("sub_record2")
		strName := "1"
		So(rec, ShouldResemble, &Record{
			kind:  "record",
			Name:  "foo",
			Slice: []string{"foo", "bar"},
			Sub: &SubRecord2{
				kind:    "sub_record2",
				PtrName: &encapsulated,
				Name:    encapsulated,
				Subs: []SubRecord{
					SubRecord{
						kind: "sub_record",
						Name: &strName,
					},
				},
			},
		})
	})
	Convey("Decode a nested object, unexpected/misspelled fields", t, func() {
		m := map[string]interface{}{
			"name":  "foo",
			"kind":  "record",
			"slice": []string{"foo", "bar"},
			"sub": map[string]interface{}{
				"subs": []map[string]interface{}{
					{
						"kind": "sub_record",
						"name": "1",
					},
				},
				"kind":    "sub_record2",
				"ptrname": "sub_record2",
				"namer":   "sub_record2",
			},
		}
		dec, err := decode.Decode(m, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		strName := "1"
		So(rec, ShouldResemble, &Record{
			kind:  "record",
			Name:  "foo",
			Slice: []string{"foo", "bar"},
			Sub: &SubRecord2{
				kind: "sub_record2",
				Subs: []SubRecord{
					SubRecord{
						kind: "sub_record",
						Name: &strName,
					},
				},
			},
		})
	})
	Convey("Unmarshal JSON of a nested object", t, func() {
		b, err := json.Marshal(m)
		So(err, ShouldBeNil)
		dec, err := decode.UnmarshalJSON(b, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		name := "bar"
		So(rec, ShouldResemble, &Record{
			kind:  "record",
			Name:  "foo",
			Slice: []string{"foo", "bar"},
			Sub:   &SubRecord{kind: "sub_record", Name: &name},
		})
	})
	Convey("Unmarshal bad JSON", t, func() {
		b, err := json.Marshal(m)
		So(err, ShouldBeNil)
		_, err = decode.UnmarshalJSON(b[1:], "kind", MyTestFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("Test decoding - bad payload - array when array is not expected", t, func() {
		m := map[string]interface{}{
			"name": []map[string]interface{}{
				{
					"kind": "sub_record",
					"name": "1",
				},
			},
		}
		_, err := decode.DecodeInto(m, &PetOwner{}, SchemaPathFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("Test parsing and decoding - bad payload - object when object is not expected", t, func() {
		b := `{ "name": { "type": "Palace"}}`
		_, err := decode.UnmarshalJSONInto([]byte(b), &PetOwner{}, SchemaPathFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("Test OneOf decoding - pets1.json", t, func() {
		// load spec from testdata identified by file
		bytes, err := ioutil.ReadFile("testdata/pets1.json")
		So(err, ShouldBeNil)

		v, err := decode.UnmarshalJSONInto(bytes, &Envelope{}, SchemaPathFactory)
		So(err, ShouldBeNil)
		_, err = json.MarshalIndent(v, "", "  ")
		So(err, ShouldBeNil)
	})
	Convey("Test OneOf decoding - array of objects", t, func() {
		b := `{ "name": "john", "owns": [{ "type": "Palace"}, {"type": "House"}]}`
		_, err := decode.UnmarshalJSONInto([]byte(b), &PetOwner{}, SchemaPathFactory)
		So(err, ShouldBeNil)
	})
	Convey("Test OneOf decoding - array of user crafted objects ", t, func() {
		var y = struct {
			LivesIn *[]struct{ Age *int }
		}{}
		var x = struct {
			LivesIn []*struct{ Age *int }
		}{}
		var z = struct {
			LivesIn []*struct{ Age int }
		}{}
		m := map[string]interface{}{
			"livesIn": []map[string]interface{}{
				{"age": 7},
			},
		}

		_, err := decode.DecodeInto(m, &y, SchemaPathFactory)
		So(err, ShouldBeNil)
		_, err = decode.DecodeInto(m, &x, SchemaPathFactory)
		So(err, ShouldBeNil)
		_, err = decode.DecodeInto(m, &z, SchemaPathFactory)
		So(err, ShouldBeNil)
	})
	Convey("Test OneOf decoding - array of objects - bad oneOf", t, func() {

		b := `{ "name": "john", "owns": [{ "class": "Palace"}, {"class":12}]}`
		_, err := decode.UnmarshalJSONInto([]byte(b), &PetOwner{}, SchemaPathFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("Test OneOf decoding - array of objects - bad property type", t, func() {
		b := `{ "name": "john", "owns": [{ "class": { "type": "House", "rooms": "string"}}]}`
		_, err := decode.UnmarshalJSONInto([]byte(b), &PetOwner{}, SchemaPathFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("Test OneOf decoding - wrong oneOf discriminator", t, func() {
		b := `{ "name": "john", "livesIn": { "class": "Palace"}}`
		_, err := decode.UnmarshalJSONInto([]byte(b), &PetOwner{}, SchemaPathFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("Test OneOf decoding - invalid discriminator value", t, func() {
		b := `{ "name": "john", "livesIn": {"type": "car"} }`
		_, err := decode.UnmarshalJSONInto([]byte(b), &PetOwner{}, SchemaPathFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("Test OneOf decoding - bad json", t, func() {
		b := `{ "name": `
		_, err := decode.UnmarshalJSONInto([]byte(b), &PetOwner{}, SchemaPathFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("Test OneOf decoding - can decode into object that has required array", t, func() {
		x := LivesInRequiredArray{}
		b := `{ "livesIn": [ "class", "Palace"]}`
		i, err := decode.UnmarshalJSONInto([]byte(b), &x, SchemaPathFactory)
		So(err, ShouldBeNil)
		So(i.(*LivesInRequiredArray), ShouldResemble, &LivesInRequiredArray{LivesIn: []string{"class", "Palace"}})
	})
	Convey("Test OneOf decoding - can decode into object that has required array with nested object", t, func() {
		x := LivesInRequiredArrayNested{}
		b := `{ "livesIn": [ { "age":7, "name": "spot", "lost": false } ] }`
		i, err := decode.UnmarshalJSONInto([]byte(b), &x, SchemaPathFactory)
		So(err, ShouldBeNil)
		So(i.(*LivesInRequiredArrayNested), ShouldResemble, &LivesInRequiredArrayNested{LivesIn: []RequiredBasicTypes{{Age: 7, Name: "spot", Lost: false}}})
	})
	Convey("Test OneOf decoding - can decode into object that has an array of pointers", t, func() {
		x := LivesInArrayOfPointers{}
		b := `{ "livesIn": [ "class", "Palace" ]}`
		i, err := decode.UnmarshalJSONInto([]byte(b), &x, SchemaPathFactory)
		So(err, ShouldBeNil)
		string1 := "class"
		string2 := "Palace"
		So(i.(*LivesInArrayOfPointers), ShouldResemble, &LivesInArrayOfPointers{LivesIn: []*string{&string1, &string2}})
	})
	Convey("Test OneOf decoding - can decode into object that has required basic types", t, func() {
		y := LivesInStruct{}
		b := `{ "livesIn": { "age": 7, "name": "spot", "lost": false}}`
		i, err := decode.UnmarshalJSONInto([]byte(b), &y, SchemaPathFactory)
		So(err, ShouldBeNil)
		So(i.(*LivesInStruct), ShouldResemble, &LivesInStruct{LivesIn: &RequiredBasicTypes{Age: 7, Name: "spot", Lost: false}})
	})
	Convey("Test OneOf decoding - cannot decode into object is not struct pointer", t, func() {
		b := `{ "name": "john"}`
		i := 1
		_, err := decode.UnmarshalJSONInto([]byte(b), i, SchemaPathFactory)
		So(err, ShouldNotBeNil)
		_, err = decode.UnmarshalJSONInto([]byte(b), &i, SchemaPathFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("Test OneOf decoding - invalid oneOf field kind", t, func() {
		b := `{ "name": "john", "livesIn": [] }`
		_, err := decode.UnmarshalJSONInto([]byte(b), &PetOwner{}, SchemaPathFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("Test parsing and decoding with underlying type", t, func() {
		b := `{ "name": "john", "updateTime": "2019-10-21T14:56:28.292468-06:00" }`
		i, err := decode.UnmarshalJSONInto([]byte(b), &TimedStruct{}, nil)
		So(err, ShouldBeNil)
		expTime, _ := time.Parse(time.RFC3339, "2019-10-21T14:56:28.292468-06:00")
		So(i.(*TimedStruct), ShouldResemble, &TimedStruct{Name: "john", UpdateTime: &expTime})

	})
	Convey("Test parsing and decoding with incorrect underlying type", t, func() {
		b := `{ "name": "john", "updateTime": "string" }`
		_, err := decode.UnmarshalJSONInto([]byte(b), &TimedStruct{}, nil)
		So(err, ShouldNotBeNil)
	})
	Convey("Test parsing and decoding with incorrect field type in payload", t, func() {
		b := `{ "name": { "type": "john"}, "updateTime": "string" }`
		_, err := decode.UnmarshalJSONInto([]byte(b), &TimedStruct{}, testSPF)
		So(err, ShouldNotBeNil)
	})
	Convey("Test decoding -- cannot decode when required field is set to null", t, func() {
		b := `{ "name": null, "updateTime": "2019-10-21T14:56:28.292468-06:00"}`
		_, err := decode.UnmarshalJSONInto([]byte(b), &TimedStruct{}, SchemaPathFactory)
		So(err, ShouldNotBeNil)
	})
	Convey("Test decoding -- allow decode when unrequired field is set to null", t, func() {
		b := `{ "name": "john", "updateTime": null }`
		i, err := decode.UnmarshalJSONInto([]byte(b), &TimedStruct{}, SchemaPathFactory)
		So(err, ShouldBeNil)
		So(i.(*TimedStruct), ShouldResemble, &TimedStruct{Name: "john"})
	})
}

// Test that fields for which there's no value in a payload are still set from defaults specified in struct tags
// *Note* the tests here MUST match the contents of the tags specified in the struct variable up the top of the file
func TestDefaultValues(t *testing.T) {

	m := map[string]interface{}{}
	Convey("Setting All Defaults", t, func() {
		swd := &StructWithDefaults{}
		_, err := decode.DecodeIntoWithDefaults(m, swd, testSPF, true)
		So(err, ShouldBeNil)
		So(swd.Int, ShouldEqual, -7)
		So(swd.UInt, ShouldEqual, 12)
		So(swd.Val, ShouldEqual, "STRING_VAL")
		So(swd.Ptr, ShouldNotBeNil)
		sp := "STRING_PTR"
		So(swd.Ptr, ShouldResemble, &sp)
		tm, _ := time.Parse(time.RFC3339, "2019-10-28T12:35:56Z")
		So(swd.Time, ShouldEqual, tm)
		tm, _ = time.Parse(time.RFC3339, "2019-10-28T23:45:10Z")
		So(swd.PTime, ShouldResemble, &tm)

		So(swd.Int8, ShouldEqual, 127)
		So(swd.UInt8, ShouldEqual, 127)
		So(swd.Int16, ShouldEqual, 32767)
		So(swd.UInt16, ShouldEqual, 65535)
		So(swd.Int32, ShouldEqual, 2147483647)
		So(swd.UInt32, ShouldEqual, 4294967295)
		So(swd.Int64, ShouldEqual, 32767)
		So(swd.UInt64, ShouldEqual, 65535)

		So(swd.Float32, ShouldEqual, 1.0)
		So(swd.Float64, ShouldEqual, 1.0)
		So(swd.Bool, ShouldEqual, true)

		// no payload, so SR.Name should be nil
		So(swd.SR.Name, ShouldBeNil)

	})

	m = map[string]interface{}{"SR": map[string]interface{}{}}
	Convey("Setting Nested field Defaults", t, func() {
		swd := &StructWithDefaults{}

		_, err := decode.DecodeIntoWithDefaults(m, swd, testSPF, true)
		So(err, ShouldBeNil)

		So(swd.SR.Name, ShouldNotBeNil)
		james := "James"
		So(swd.SR.Name, ShouldResemble, &james)

	})

	Convey("Bad default values", t, func() {
		type BIS struct {
			BadIntStr int `default:"aaaa"`
		}
		type BB struct {
			BadBool bool `default:"text"`
		}
		type BF32 struct {
			BadFloat32 bool `default:"1E1234567"`
		}
		type BTC struct {
			Chan chan int `default:"1E1234567"`
		}
		type BTS struct {
			Time *time.Time `default:"--123"`
		}

		_, err := decode.DecodeIntoWithDefaults(m, &BIS{}, testSPF, true)
		So(err, ShouldNotBeNil)
		_, err = decode.DecodeIntoWithDefaults(m, &BB{}, testSPF, true)
		So(err, ShouldNotBeNil)
		_, err = decode.DecodeIntoWithDefaults(m, &BF32{}, testSPF, true)
		So(err, ShouldNotBeNil)
		_, err = decode.DecodeIntoWithDefaults(m, &BTC{}, testSPF, true)
		So(err, ShouldNotBeNil)
		_, err = decode.DecodeIntoWithDefaults(m, &BTS{}, testSPF, true)
		So(err, ShouldNotBeNil)
	})

}
