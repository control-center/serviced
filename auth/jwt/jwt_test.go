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
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	jwtgo "github.com/dgrijalva/jwt-go"
	. "gopkg.in/check.v1"
)

// This plumbs gocheck into testing
func Test(t *testing.T) {
	TestingT(t)
}

type JwtSuite struct{}

var _ = Suite(&JwtSuite{})

func (t *JwtSuite) TestEncodeAndSignToken(c *C) {
	jwt, _ := NewInstance(DEFAULT_ALGORITHM, t.getDummyKeyLookup())
	token := t.getValidToken()

	encodedToken, err := jwt.EncodeAndSignToken(token)

	// Basic sanity checks
	c.Assert(err, IsNil)
	c.Assert(encodedToken, NotNil)
	parts := strings.Split(encodedToken, ".")
	c.Assert(len(parts), Equals, 3)

	// Verify the decoded header matches the expected values
	headerBytes, err := jwtgo.DecodeSegment(parts[0])
	c.Assert(err, IsNil)
	c.Assert(headerBytes, NotNil)
	var header map[string]interface{}
	err = json.Unmarshal(headerBytes, &header)
	c.Assert(err, IsNil)
	c.Assert(header["typ"], Equals, token.Header["typ"])
	c.Assert(header["alg"], Equals, token.Header["alg"])

	// Verify the decoded Claims match the expected values
	claimsBytes, err := jwtgo.DecodeSegment(parts[1])
	c.Assert(err, IsNil)
	c.Assert(claimsBytes, NotNil)
	var claims map[string]interface{}
	err = json.Unmarshal(claimsBytes, &claims)
	c.Assert(err, IsNil)
	c.Assert(claims["iat"], Equals, token.Claims["iat"])
	c.Assert(claims["iss"], Equals, token.Claims["iss"])
	c.Assert(claims["sub"], Equals, token.Claims["sub"])
	c.Assert(claims["zav"], Equals, token.Claims["zav"])
	c.Assert(claims["req"], Equals, token.Claims["req"])

	// Verify that the signature portion is a valid encoding
	signatureBytes, err := jwtgo.DecodeSegment(parts[2])
	c.Assert(err, IsNil)
	c.Assert(signatureBytes, NotNil)
	c.Assert(len(signatureBytes), Not(Equals), 0)

	// Compute the expected signature and compare it to the actual one.
	jwtFacade, _ := jwt.(*jwtFacade)
	signerFacade, _ := jwtFacade.signer.(*signerFacade)
	signatureSource := fmt.Sprintf("%s.%s", parts[0], parts[1])
	key, _ := t.getDummyKeyLookup()(nil)
	expectedSignature, err := signerFacade.signingMethod.Sign(signatureSource, key)
	c.Assert(err, IsNil)
	c.Assert(parts[2], Equals, expectedSignature)
}

func (t *JwtSuite) TestDecodeToken(c *C) {
	jwt, _ := NewInstance(DEFAULT_ALGORITHM, t.getDummyKeyLookup())
	expectedToken := t.getValidToken()

	// Manually sign the token
	jsonHeader, _ := json.Marshal(expectedToken.Header)
	header := jwtgo.EncodeSegment(jsonHeader)
	jsonClaims, _ := json.Marshal(expectedToken.Claims)
	claims := jwtgo.EncodeSegment(jsonClaims)
	jwtFacade, _ := jwt.(*jwtFacade)
	signerFacade, _ := jwtFacade.signer.(*signerFacade)
	signatureSource := fmt.Sprintf("%s.%s", header, claims)
	key, _ := t.getDummyKeyLookup()(nil)
	signedSource, err := signerFacade.signingMethod.Sign(signatureSource, key)
	signature := fmt.Sprintf("%s.%s.%s", header, claims, signedSource)

	actualToken, err := jwt.DecodeToken(signature)

	c.Assert(err, IsNil)
	t.assertMapsEqual(c, actualToken.Header, expectedToken.Header)
	t.assertMapsEqual(c, actualToken.Claims, expectedToken.Claims)
	c.Assert(actualToken.Signature, NotNil)
	c.Assert(actualToken.Signature, Equals, strings.Split(signature, ".")[2])
}

// Verify we can eat our own dog food
func (t *JwtSuite) TestDecodeASignedToken(c *C) {
	jwt, _ := NewInstance(DEFAULT_ALGORITHM, t.getDummyKeyLookup())
	expectedToken := t.getValidToken()

	encodedToken, err := jwt.EncodeAndSignToken(expectedToken)
	c.Assert(err, IsNil)

	actualToken, err := jwt.DecodeToken(encodedToken)
	c.Assert(err, IsNil)

	t.assertMapsEqual(c, actualToken.Header, expectedToken.Header)
	t.assertMapsEqual(c, actualToken.Claims, expectedToken.Claims)
	c.Assert(actualToken.Signature, NotNil)
	c.Assert(actualToken.Signature, Equals, strings.Split(encodedToken, ".")[2])
}

func (t *JwtSuite) TestEncodeAndSignTokenFailsForEmptyToken(c *C) {
	jwt, _ := NewInstance(DEFAULT_ALGORITHM, t.getDummyKeyLookup())
	token, _ := jwt.NewToken("GET", "http://control-center/services", "", nil)
	token.Claims = make(map[string]interface{})

	encodedToken, err := jwt.EncodeAndSignToken(token)

	c.Assert(encodedToken, Equals, "")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "invalid token: token.Claims invalid: can not be empty")
}

func (t *JwtSuite) TestEncodeAndSignTokenFailsForBadKey(c *C) {
	lookupError := fmt.Errorf("force key lookup failure")
	jwt, _ := NewInstance(DEFAULT_ALGORITHM, func(claims map[string]interface{}) (interface{}, error) {
		return nil, lookupError
	})
	token := t.getValidToken()

	encodedToken, err := jwt.EncodeAndSignToken(token)

	c.Assert(encodedToken, Equals, "")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Key lookup failed: force key lookup failure")
}

func (t *JwtSuite) TestEncodeAndSignTokenFailsSigning(c *C) {
	jwt, _ := NewInstance(DEFAULT_ALGORITHM, t.getDummyKeyLookup())
	token := t.getValidToken()
	token.Header["badField"] = make(chan int)

	encodedToken, err := jwt.EncodeAndSignToken(token)

	c.Assert(encodedToken, Equals, "")
	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Error encoding token: json: unsupported type: chan int")
}

func (t *JwtSuite) TestDecodeTokenFailsForEmptySignature(c *C) {
	jwt, _ := NewInstance(DEFAULT_ALGORITHM, t.getDummyKeyLookup())

	token, err := jwt.DecodeToken("")

	c.Assert(token, IsNil)
	c.Assert(err.Error(), Equals, "Failed to decode token: token contains an invalid number of segments")
}

func (t *JwtSuite) TestDecodeTokenFailsForBadSignature(c *C) {
	jwt, _ := NewInstance(DEFAULT_ALGORITHM, t.getDummyKeyLookup())

	// Manually sign the token, but tamper with the value of the signature segment
	expectedToken := t.getValidToken()
	jsonHeader, _ := json.Marshal(expectedToken.Header)
	header := jwtgo.EncodeSegment(jsonHeader)
	jsonClaims, _ := json.Marshal(expectedToken.Claims)
	claims := jwtgo.EncodeSegment(jsonClaims)
	jwtFacade, _ := jwt.(*jwtFacade)
	signerFacade, _ := jwtFacade.signer.(*signerFacade)
	signatureSource := fmt.Sprintf("%s.%s.badSalt", header, claims)
	key, _ := t.getDummyKeyLookup()(nil)
	signedSource, err := signerFacade.signingMethod.Sign(signatureSource, key)
	encodedToken := fmt.Sprintf("%s.%s.%s", header, claims, signedSource)

	token, err := jwt.DecodeToken(encodedToken)

	c.Assert(err.Error(), Equals, "Failed to decode token: signature is invalid")
	c.Assert(token, IsNil)
}

func (t *JwtSuite) TestDecodeTokenFailsForBadKey(c *C) {
	lookupError := fmt.Errorf("force key lookup failure")
	jwt, _ := NewInstance(DEFAULT_ALGORITHM, func(claims map[string]interface{}) (interface{}, error) {
		return nil, lookupError
	})

	// Manually sign the token with a good key
	expectedToken := t.getValidToken()
	jsonHeader, _ := json.Marshal(expectedToken.Header)
	header := jwtgo.EncodeSegment(jsonHeader)
	jsonClaims, _ := json.Marshal(expectedToken.Claims)
	claims := jwtgo.EncodeSegment(jsonClaims)
	jwtFacade, _ := jwt.(*jwtFacade)
	signerFacade, _ := jwtFacade.signer.(*signerFacade)
	signatureSource := fmt.Sprintf("%s.%s", header, claims)
	key, _ := t.getDummyKeyLookup()(nil)
	signedSource, err := signerFacade.signingMethod.Sign(signatureSource, key)
	encodedToken := fmt.Sprintf("%s.%s.%s", header, claims, signedSource)

	token, err := jwt.DecodeToken(encodedToken)

	c.Assert(err.Error(), Equals, "Failed to decode token: force key lookup failure")
	c.Assert(token, IsNil)
}

func (t *JwtSuite) TestValidateToken(c *C) {
	request := t.getValidRequest()
	jwt, _ := NewInstance(DEFAULT_ALGORITHM, t.getDummyKeyLookup())
	signedToken, err := t.getSignedToken(jwt, t.getValidToken())
	c.Assert(err, IsNil)

	err = jwt.ValidateToken(signedToken, request.Method, request.URL.String(), nil, time.Duration(60)*time.Second)

	c.Assert(err, IsNil)
}

func (t *JwtSuite) TestValidateTokenFailsMinimumRequirement(c *C) {
	request := t.getValidRequest()
	jwt, _ := NewInstance(DEFAULT_ALGORITHM, t.getDummyKeyLookup())
	badToken := t.getValidToken()
	delete(badToken.Header, "typ")

	err := jwt.ValidateToken(badToken, request.Method, request.URL.String(), nil, time.Duration(60)*time.Second)

	c.Assert(err.Error(), Equals, "invalid token: token.Header invalid: missing \"typ\"")
}

func (t *JwtSuite) TestValidateTokenFailsForEmptySignature(c *C) {
	request := t.getValidRequest()
	jwt, _ := NewInstance(DEFAULT_ALGORITHM, t.getDummyKeyLookup())

	err := jwt.ValidateToken(t.getValidToken(), request.Method, request.URL.String(), nil, time.Duration(60)*time.Second)

	c.Assert(err.Error(), Equals, "token is missing Signature")
}

func (t *JwtSuite) TestValidateTokenFailsForInvalidIAT(c *C) {
	request := t.getValidRequest()
	jwt, _ := NewInstance(DEFAULT_ALGORITHM, t.getDummyKeyLookup())
	minToken := t.getValidToken()
	minToken.Claims["iat"] = true
	signedToken, err := t.getSignedToken(jwt, minToken)
	c.Assert(err, IsNil)

	err = jwt.ValidateToken(signedToken, request.Method, request.URL.String(), nil, time.Duration(60)*time.Second)

	c.Assert(err.Error(), Equals, "Claims['iat'] is not valid: Type \"bool\" is not valid")
}

func (t *JwtSuite) TestValidateTokenFailsForExpiredIAT(c *C) {
	request := t.getValidRequest()
	expirationLimt := time.Duration(60) * time.Second
	jwt, _ := NewInstance(DEFAULT_ALGORITHM, t.getDummyKeyLookup())
	minToken := t.getValidToken()
	minToken.Claims["iat"] = float64(time.Now().Unix() - int64(expirationLimt) - 1)
	signedToken, err := t.getSignedToken(jwt, minToken)
	c.Assert(err, IsNil)

	err = jwt.ValidateToken(signedToken, request.Method, request.URL.String(), nil, expirationLimt)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "token has expired")
}

func (t *JwtSuite) TestValidateTokenIgnoresIAT(c *C) {
	var expirationLimt float64
	request := t.getValidRequest()
	jwt, _ := NewInstance(DEFAULT_ALGORITHM, t.getDummyKeyLookup())
	minToken := t.getValidToken()
	minToken.Claims["iat"] = float64(time.Now().Unix() - int64(expirationLimt) - 1)
	signedToken, err := t.getSignedToken(jwt, minToken)
	c.Assert(err, IsNil)

	err = jwt.ValidateToken(signedToken, request.Method, request.URL.String(), nil, 0)

	c.Assert(err, IsNil)
}

func (t *JwtSuite) TestValidateTokenFailsWithTamperedBody(c *C) {
	body := "{\"param1\": \"value1\", \"param2\": 5}"
	request := t.getValidRequestWithBody(body)
	jwt, _ := NewInstance(DEFAULT_ALGORITHM, t.getDummyKeyLookup())
	token := t.getValidToken()
	signedToken, err := t.getSignedToken(jwt, token)
	c.Assert(err, IsNil)

	tamperedBody := strings.Replace(body, "value1", "value2", -1)
	err = jwt.ValidateToken(signedToken, request.Method, request.URL.String(), []byte(tamperedBody), 0)

	c.Assert(err, NotNil)
}

func (t *JwtSuite) TestValidateMinimumRequirementsPasses(c *C) {
	err := validateMinimumRequirements(t.getValidToken())
	c.Assert(err, IsNil)
}

func (t *JwtSuite) TestValidateMinimumRequirementsFailsForNilToken(c *C) {
	err := validateMinimumRequirements(nil)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "nil token is invalid")
}

func (t *JwtSuite) TestValidateMinimumRequirementsFailsForNilHeader(c *C) {
	token := &Token{
		Header: nil,
		Claims: t.getValidClaims(),
	}

	err := validateMinimumRequirements(token)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "token.Header invalid: can not be nil")
}

func (t *JwtSuite) TestValidateMinimumRequirementsFailsForEmptyHeader(c *C) {
	token := &Token{
		Header: make(map[string]interface{}),
		Claims: t.getValidClaims(),
	}

	err := validateMinimumRequirements(token)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "token.Header invalid: can not be empty")
}

func (t *JwtSuite) TestValidateMinimumRequirementsFailsForHeaderMissingAlg(c *C) {
	token := &Token{
		Header: map[string]interface{}{
			"typ": "JWT",
		},
		Claims: t.getValidClaims(),
	}

	err := validateMinimumRequirements(token)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "token.Header invalid: missing \"alg\"")
}

func (t *JwtSuite) TestValidateMinimumRequirementsFailsForHeaderInvalidAlg(c *C) {
	token := &Token{
		Header: map[string]interface{}{
			"typ": "JWT",
			"alg": "ponzi-scheme",
		},
		Claims: t.getValidClaims(),
	}

	err := validateMinimumRequirements(token)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "token.Header invalid: [\"alg\"] should be \"HS256\", not \"ponzi-scheme\"")
}

func (t *JwtSuite) TestValidateMinimumRequirementsFailsForHeaderMissingTyp(c *C) {
	token := &Token{
		Header: map[string]interface{}{
			"alg": DEFAULT_ALGORITHM,
		},
		Claims: t.getValidClaims(),
	}

	err := validateMinimumRequirements(token)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "token.Header invalid: missing \"typ\"")
}

func (t *JwtSuite) TestValidateMinimumRequirementsFailsForHeaderInvalidTyp(c *C) {
	token := &Token{
		Header: map[string]interface{}{
			"typ": "surfboard",
			"alg": DEFAULT_ALGORITHM,
		},
		Claims: t.getValidClaims(),
	}

	err := validateMinimumRequirements(token)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "token.Header invalid: [\"typ\"] should be \"JWT\", not \"surfboard\"")
}

func (t *JwtSuite) TestValidateMinimumRequirementsFailsForNilClaims(c *C) {
	token := &Token{
		Header: t.getValidHeader(),
		Claims: nil,
	}

	err := validateMinimumRequirements(token)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "token.Claims invalid: can not be nil")
}

func (t *JwtSuite) TestValidateMinimumRequirementsFailsForEmptyClaims(c *C) {
	token := &Token{
		Header: t.getValidHeader(),
		Claims: make(map[string]interface{}),
	}

	err := validateMinimumRequirements(token)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "token.Claims invalid: can not be empty")
}

func (t *JwtSuite) TestValidateMinimumRequirementsFailsForClaimsMissingIat(c *C) {
	claims := t.getValidClaims()
	delete(claims, "iat")
	token := &Token{
		Header: t.getValidHeader(),
		Claims: claims,
	}

	err := validateMinimumRequirements(token)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "token.Claims invalid: missing \"iat\"")
}

func (t *JwtSuite) TestValidateMinimumRequirementsFailsForClaimsMissingIss(c *C) {
	claims := t.getValidClaims()
	delete(claims, "iss")
	token := &Token{
		Header: t.getValidHeader(),
		Claims: claims,
	}

	err := validateMinimumRequirements(token)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "token.Claims invalid: missing \"iss\"")
}

func (t *JwtSuite) TestValidateMinimumRequirementsFailsForClaimsMissingSub(c *C) {
	claims := t.getValidClaims()
	delete(claims, "sub")
	token := &Token{
		Header: t.getValidHeader(),
		Claims: claims,
	}

	err := validateMinimumRequirements(token)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "token.Claims invalid: missing \"sub\"")
}

func (t *JwtSuite) TestValidateMinimumRequirementsFailsForClaimsMissingZav(c *C) {
	claims := t.getValidClaims()
	delete(claims, "zav")
	token := &Token{
		Header: t.getValidHeader(),
		Claims: claims,
	}

	err := validateMinimumRequirements(token)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "token.Claims invalid: missing \"zav\"")
}

func (t *JwtSuite) TestValidateMinimumRequirementsFailsForClaimsMissingReq(c *C) {
	claims := t.getValidClaims()
	delete(claims, "req")
	token := &Token{
		Header: t.getValidHeader(),
		Claims: claims,
	}

	err := validateMinimumRequirements(token)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "token.Claims invalid: missing \"req\"")
}

func (t *JwtSuite) TestGetIssuedAtTimeForValidString(c *C) {
	token := t.getValidToken()
	token.Claims["iat"] = "25"

	iat, err := getIssuedAtTime(token)

	c.Assert(err, IsNil)
	c.Assert(iat, Equals, 25.0)
}

func (t *JwtSuite) TestGetIssuedAtTimeForInvalidString(c *C) {
	token := t.getValidToken()
	token.Claims["iat"] = "twenty-five"

	iat, err := getIssuedAtTime(token)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "strconv.ParseFloat: parsing \"twenty-five\": invalid syntax")
	c.Assert(iat, Equals, 0.0)
}

func (t *JwtSuite) TestGetIssuedAtTimeForInvalidType(c *C) {
	token := t.getValidToken()
	token.Claims["iat"] = make(chan int)

	iat, err := getIssuedAtTime(token)

	c.Assert(err, NotNil)
	c.Assert(err.Error(), Equals, "Type \"chan int\" is not valid")
	c.Assert(iat, Equals, 0.0)
}

func (t *JwtSuite) TestGetIssuedAtTimeForFloat32(c *C) {
	var issuedAtTime float32
	issuedAtTime = 25.0
	token := t.getValidToken()
	token.Claims["iat"] = issuedAtTime

	iat, err := getIssuedAtTime(token)

	c.Assert(err, IsNil)
	c.Assert(iat, Equals, 25.0)
}

func (t *JwtSuite) TestGetIssuedAtTimeForFloat64(c *C) {
	var issuedAtTime float64
	issuedAtTime = 25.0
	token := t.getValidToken()
	token.Claims["iat"] = issuedAtTime

	iat, err := getIssuedAtTime(token)

	c.Assert(err, IsNil)
	c.Assert(iat, Equals, 25.0)
}

func (t *JwtSuite) TestGetIssuedAtTimeForInt(c *C) {
	token := t.getValidToken()
	token.Claims["iat"] = 25

	iat, err := getIssuedAtTime(token)

	c.Assert(err, IsNil)
	c.Assert(iat, Equals, 25.0)
}

func (t *JwtSuite) getDummyKeyLookup() KeyLookupFunc {
	return func(claims map[string]interface{}) (interface{}, error) {
		return []byte("somekey"), nil
	}
}

func (t *JwtSuite) getValidHeader() map[string]interface{} {
	return map[string]interface{}{
		"typ": "JWT",
		"alg": DEFAULT_ALGORITHM,
	}
}

func (t *JwtSuite) getValidClaims() map[string]interface{} {
	validRequest := t.getValidRequest()
	canonicalUrl := fmt.Sprintf("%s %s  ", validRequest.Method, validRequest.URL.Path)
	requestHash := sha256.Sum256([]byte(canonicalUrl))
	encodedHash := hex.EncodeToString(requestHash[:])

	return map[string]interface{}{
		"iat": float64(time.Now().Unix()), // use float64 to match default behavior of json.Unmarshall()
		"iss": "serviced_delegate",
		"sub": "somePoolId",
		"zav": "cc_delegate_auth_v1",
		"req": encodedHash,
	}
}

func (t *JwtSuite) getValidToken() *Token {
	token := &Token{
		Header: t.getValidHeader(),
		Claims: t.getValidClaims(),
	}
	return token
}

func (t *JwtSuite) getSignedToken(jwt JWT, unsignedToken *Token) (*Token, error) {
	encodedToken, err := jwt.EncodeAndSignToken(unsignedToken)
	if err != nil {
		return nil, err
	}

	signedToken, err := jwt.DecodeToken(encodedToken)
	if err != nil {
		return nil, err
	}
	return signedToken, err
}

func (t *JwtSuite) getValidRequest() *http.Request {
	req, _ := http.NewRequest("GET", "http://control-center/dummy", nil)
	return req
}

func (t *JwtSuite) getValidRequestWithBody(body string) *http.Request {
	req, _ := http.NewRequest("GET", "http://control-center/dummy", strings.NewReader(body))
	return req
}

func (t *JwtSuite) assertMapsEqual(c *C, map1, map2 map[string]interface{}) {
	c.Assert(map1, NotNil)
	c.Assert(map2, NotNil)
	c.Assert(len(map1), Equals, len(map2))

	for key, value1 := range map1 {
		value2, ok := map2[key]
		c.Assert(ok, Equals, true)
		c.Assert(value1, Equals, value2)
	}
}
