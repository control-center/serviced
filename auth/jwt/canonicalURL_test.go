// Copyright 2015 The Serviced Authors.
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.
package jwt

import (
	"fmt"
	"net/url"

	. "gopkg.in/check.v1"
)

type canonicalURLTest struct {
	description string
	path        string
	result      string
}

func (t *JwtSuite) TestCanonicalURL(c *C) {
	testData := []canonicalURLTest{
		{"simple", "/services", "/services  "},
		{"simple with sorted queries", "/services?q2=a&q1=a", "/services q1=a&q2=a "},
		{"sort same query params", "/services?q1=z&q1=a", "/services q1=a&q1=z "},
		{"sorted same query params w/null-value", "/services?q1=z&q1=a&q1=", "/services q1=&q1=a&q1=z "},
		{"normalize null-value params", "/services?q2&q1", "/services q1=&q2= "},
		{"null-key values first", "/services?q1=a&=null", "/services =null&q1=a "},

		{"remove duplicate slashes", "/services//someID/of//something", "/services/someID/of/something  "},
		{"remove empty query separator", "/services?", "/services  "},
		{"remove trailing slash", "/services/", "/services  "},
		{"remove dot segments 1", "/services/./something/./someID", "/services/something/someID  "},
		{"remove dot segments 2", "/services/./something/.someID", "/services/something/.someID  "},
		{"remove dot segments 3", "/services/../someID", "/someID  "},
		{"remove dot segments 4", "/services/../../someID", "/someID  "},
		{"decode unnecessary escapes", "/services/%41%42%43/%30%31%32?foo=%33", "/services/ABC/012 foo=3 "},
	}

	for _, data := range testData {
		rawURL := fmt.Sprintf("http://control-center%s", data.path)

		canonical, err := CanonicalURL("GET", rawURL, "", nil)

		c.Assert(err, IsNil)
		c.Assert(string(canonical), Equals, fmt.Sprintf("GET %s", data.result))
	}
}

func (t *JwtSuite) TestCanonicalURLEdgeCases(c *C) {
	testData := []canonicalURLTest{
		// instead of converting mixed-case values for necessary escapes,
		//		golang's url.Parse() is actually unescaping them :-(
		{"expected /services/%2A%3B%24", "/services/%2a%3b%24", "/services/*;$  "},

		// golang's url.Parse() removes fragments since these aren't sent to servers
		{"fragments are removed", "/services/#frag", "/services  "},

		// golang's url.Parse() unescapes some reserved percent-encoded values in query params
		{"expected /services q1=a%2F%26b&q2=x", "/services?q1=a%2f%26b&q2=x", "/services q1=a/%26b&q2=x "},
	}

	for _, data := range testData {
		rawURL := fmt.Sprintf("http://control-center%s", data.path)

		canonical, err := CanonicalURL("GET", rawURL, "", nil)

		c.Assert(err, IsNil)
		c.Assert(string(canonical), Equals, fmt.Sprintf("GET %s", data.result))
	}
}

func (t *JwtSuite) TestCanonicalURLWithBody(c *C) {
	body := "{\"param1\": \"value1\", \"param2\": 5}"

	canonical, err := CanonicalURL("GET", "http://control-center/services", "", []byte(body))

	c.Assert(err, IsNil)
	c.Assert(string(canonical), Equals, fmt.Sprintf("GET /services  %s", body))
}

func (t *JwtSuite) TestCanonicalURLWithInvalidURL(c *C) {
	invalidURL := "ba!@#$%^://control-center"

	canonical, err := CanonicalURL("GET", invalidURL, "", nil)

	c.Assert(err, NotNil)
	c.Assert(err, ErrorMatches, "failed to parse input string.*")
	c.Assert(canonical, IsNil)
}

func (t *JwtSuite) TestGetCanonicalURIWithPrefix(c *C) {

	url, _ := url.Parse("http://control-center/dummy/services")
	uriPrefix := "/dummy"

	canonical := getCanonicalURI(url, uriPrefix)

	c.Assert(string(canonical), Equals, "/services")
}

func (t *JwtSuite) TestGetCanonicalURIWithoutPrefix(c *C) {

	url, _ := url.Parse("http://control-center/dummy/services")
	uriPrefix := ""

	canonical := getCanonicalURI(url, uriPrefix)

	c.Assert(canonical, Equals, "/dummy/services")
}

func (t *JwtSuite) TestGetCanonicalURIWithPrefixAndTrailingSlash(c *C) {

	url, _ := url.Parse("http://control-center/dummy/services/")
	uriPrefix := "/dummy/"

	canonical := getCanonicalURI(url, uriPrefix)

	c.Assert(canonical, Equals, "/services")
}

func (t *JwtSuite) TestGetCanonicalURIWhenPrefixMatchesPath(c *C) {

	url, _ := url.Parse("http://control-center/services")
	uriPrefix := "/services"

	canonical := getCanonicalURI(url, uriPrefix)

	c.Assert(canonical, Equals, "/")
}

func (t *JwtSuite) TestGetCanonicalURIWhenPrefixDoesNotMatchPath(c *C) {

	url, _ := url.Parse("http://control-center/services")
	uriPrefix := "/servicesss"

	canonical := getCanonicalURI(url, uriPrefix)

	c.Assert(canonical, Equals, "/services")
}

func (t *JwtSuite) TestGetCanonicalURIForEmptyPath(c *C) {

	url, _ := url.Parse("http://control-center") // w/o trailing slash
	canonical := getCanonicalURI(url, "")
	c.Assert(canonical, Equals, "/")

	url, _ = url.Parse("http://control-center/") // w/trailing slash
	canonical = getCanonicalURI(url, "")
	c.Assert(canonical, Equals, "/")

	url, _ = url.Parse("http://control-center/")
	canonical = getCanonicalURI(url, "/")
	c.Assert(canonical, Equals, "/")
}
