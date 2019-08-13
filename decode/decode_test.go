package decode_test

import (
	"testing"
	"fmt"
	"encoding/json"

	. "github.com/smartystreets/goconvey/convey"
	"github.com/weberr13/go-decode/decode"
)

type SubRecord struct {
	kind string		
	Name string		
}

func (r SubRecord) Kind() string {
	return r.kind
}

func NewSubRecord() decode.Decodeable {
	return &SubRecord{
		kind: "sub_record",
	}
}

type SubRecord2 struct {
	kind string		
	Name string
	Subs []SubRecord	
}

func (r SubRecord2) Kind() string {
	return r.kind
}

func NewSubRecord2() decode.Decodeable {
	return &SubRecord2{
		kind: "sub_record2",
	}
}

type Record struct {
	kind string		 
	Name string	 
	Optional *string
	Num *int
	Slice []string
	Sub  decode.Decodeable
}

func (r Record) Kind() string {
	return r.kind
}

func NewRecord() decode.Decodeable {
	return &Record{
		kind: "record",
	}
}

func MyTestFactory(kind string) (decode.Decodeable, error) {
	fm := map[string]func() decode.Decodeable {
		"record": NewRecord,
		"sub_record": NewSubRecord,
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
		"name": "foo",
		"kind": "record",
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
	Convey("unrully child object", t, func() {
		mp := map[string]interface{}{
			"name": "foo",
			"kind": "record",
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
			"name": "foo",
			"kind": "record",
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
			"name": "foo",
			"kind": "record",
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
	Convey("Decode a nested object", t, func(){
		dec, err := decode.Decode(m, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		So(dec.Kind(), ShouldEqual, "record")
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		So(rec.Sub.Kind(), ShouldEqual, "sub_record")
		So(rec, ShouldResemble, &Record{kind: "record", Name: "foo", Slice: []string{"foo", "bar"}, Sub: &SubRecord{kind: "sub_record", Name: "bar"}})
	})	
	Convey("Unmarshal a nested object, different subtype", t, func(){
		m := map[string]interface{}{
			"name": "foo",
			"kind": "record",
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
		b, err := json.Marshal(m)
		So(err, ShouldBeNil)
		dec, err := decode.UnmarshalJSON(b, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		So(dec.Kind(), ShouldEqual, "record")
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		So(rec.Sub.Kind(), ShouldEqual, "sub_record2")
		So(rec, ShouldResemble, &Record{
			kind: "record", 
			Name: "foo", 
			Slice: []string{"foo", "bar"}, 
			Sub: &SubRecord2{
				kind: "sub_record2", 
				Subs: []SubRecord{
					SubRecord{
						kind: "sub_record",
						Name: "1",
					},
				},
			},
		})
	})
	Convey("Decode a nested object, different subtype", t, func(){
		m := map[string]interface{}{
			"name": "foo",
			"kind": "record",
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
		So(dec.Kind(), ShouldEqual, "record")
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		So(rec.Sub.Kind(), ShouldEqual, "sub_record2")
		So(rec, ShouldResemble, &Record{
			kind: "record", 
			Name: "foo", 
			Slice: []string{"foo", "bar"}, 
			Sub: &SubRecord2{
				kind: "sub_record2", 
				Subs: []SubRecord{
					SubRecord{
						kind: "sub_record",
						Name: "1",
					},
				},
			},
		})
	})
	Convey("Unmarshal JSON of a nested object", t, func(){
		b, err := json.Marshal(m)
		So(err, ShouldBeNil)
		dec, err := decode.UnmarshalJSON(b, "kind", MyTestFactory)
		So(err, ShouldBeNil)
		So(dec.Kind(), ShouldEqual, "record")
		rec, ok := dec.(*Record)
		So(ok, ShouldBeTrue)
		So(rec.Sub.Kind(), ShouldEqual, "sub_record")
		So(rec, ShouldResemble, &Record{kind: "record", Name: "foo", Slice: []string{"foo", "bar"}, Sub: &SubRecord{kind: "sub_record", Name: "bar"}})
	})
	Convey("Unmarshal bad JSON", t, func(){
		b, err := json.Marshal(m)
		So(err, ShouldBeNil)
		_, err = decode.UnmarshalJSON(b[1:], "kind", MyTestFactory)
		So(err, ShouldNotBeNil)
	})
}