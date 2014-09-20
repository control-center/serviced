package commons

import (
	"fmt"
	"testing"
)

type testObject struct {
	i int
}

func TestInvalidateCache(t *testing.T) {
	c := NewCache()
	orig := testObject{}

	c.data["abc"] = orig
	c.Invalidate("abc")

	if _, ok := c.data["abc"]; ok {
		t.Fatalf("Invalidate() didn't invalidate")
	}
}

func TestAddToCache(t *testing.T) {
	c := NewCache()
	var (
		orig interface{}
	)
	obj := testObject{4}
	orig = obj

	c.Add("abc", orig)
	val, ok := c.data["abc"]
	if !ok {
		t.Fatalf("Add() didn't add to the cache")
	}
	if &val == &orig {
		t.Fatalf("Add() didn't keep a copy")
	}
}

func TestGetFromCache(t *testing.T) {
	c := NewCache()
	var orig interface{}
	orig = testObject{}

	c.data["abc"] = orig
	val, ok := c.Get("abc")
	if !ok {
		t.Fatalf("Couldn't Get() from the cache")
	}
	if &val == &orig {
		t.Fatalf("Get() didn't return a copy")
	}
	val, ok = c.Get("cde")
	if ok {
		t.Fatalf("Get()ting an invalid cache item was ok")
	}
}

func TestGetInto(t *testing.T) {
	c := NewCache()
	var source, target interface{}

	source = testObject{5}
	target = testObject{}

	c.Add("abc", source)
	ok, _ := c.GetInto("abc", &target)
	if !ok {
		t.Fatalf("Couldn't GetInto() a valid cache item")
	}
	isto, ok := target.(testObject)
	if isto.i != 5 {
		fmt.Printf("%+v %+v", isto, source)
		t.Fatalf("GetInto() target doesn't match original")
	}
}
