// Copyright 2014 The Serviced Authors.
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

package proxy

import (
	"fmt"
	"io/ioutil"
)

var (
	// command to generate: openssl req -x509 -sha256 -nodes -days 1826 -newkey rsa:2048 -keyout NEW_SERVER_KEY.key -out NEW_SERVER_CERT.crt
	InsecureCertPEM = `-----BEGIN CERTIFICATE-----
MIIDUTCCAjmgAwIBAgIJAN0kmDdJoXoNMA0GCSqGSIb3DQEBCwUAMD8xCzAJBgNV
BAYTAlVTMQ4wDAYDVQQIDAVUZXhhczEPMA0GA1UEBwwGQXVzdGluMQ8wDQYDVQQK
DAZaZW5vc3MwHhcNMTYwMTE4MjEwNjA0WhcNMjEwMTE3MjEwNjA0WjA/MQswCQYD
VQQGEwJVUzEOMAwGA1UECAwFVGV4YXMxDzANBgNVBAcMBkF1c3RpbjEPMA0GA1UE
CgwGWmVub3NzMIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAr1uvz01/
9mX0CwUcXbtMxmuhiqNXG6yHVw5EtMpMvt+NcXJ1G1USyc5BIdYRFzQft9Gy6fku
NU1XLLE33YEJouA0s0QQGxdEeO8XyWYcSIBhHYe281forXcuIMbQRIYjB6SWVp7y
espXR9u8JNUK5z9WGoyV0Dfc6HW/zUVtYxSzGQV7itJh9ehwRTfRqghyEA4q2Bc6
QseoMM4zmqn+57TX9n9VwDfIZef2N0uhZGWlMmcjdZCEzyAEOMMOq/UTg/0YmHR7
+4GHsCFexAAFUakkAAZEWJRqznG6ESjJ4HmFRhxV5SasbG6XBs7W443/6XEcZN2O
roW9kplT299srwIDAQABo1AwTjAdBgNVHQ4EFgQUc3Ei8Sngu09d6HdZcXtjdG66
3AswHwYDVR0jBBgwFoAUc3Ei8Sngu09d6HdZcXtjdG663AswDAYDVR0TBAUwAwEB
/zANBgkqhkiG9w0BAQsFAAOCAQEAC1fdEwJ4kKpB98FsVbnQrhMvbSAgh9bsRgPY
RSokHBKIEIQp7poGj0lRgd5lb97d5BfdbN6e6AO7QBGZTAz5udRQfJYWfdPkFOKg
CGjCl7QwxCN5rXBnRU39ovWaDbWMDFPSZWI3rSCFNgXi7aEYa2lY3nvst/bMBgP/
IAMQcVeLHKSlyPrT3rxiZfsQuirjLCFpsJCV4vPMPmQTOuqpJwwfDOZKqL32Y4V5
zAfukaBSHiPViIiqlufhk75Bctx1YFWyO3YK4SaJhVHXGhyXRY5yFLjWyWy+4gRg
fKTDdkaRWpMPOXGzGTwRi3bI/zDNG7NvAJg8GfUtloDiJUvf+w==
-----END CERTIFICATE-----`

	InsecureKeyPEM = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQCvW6/PTX/2ZfQL
BRxdu0zGa6GKo1cbrIdXDkS0yky+341xcnUbVRLJzkEh1hEXNB+30bLp+S41TVcs
sTfdgQmi4DSzRBAbF0R47xfJZhxIgGEdh7bzV+itdy4gxtBEhiMHpJZWnvJ6yldH
27wk1QrnP1YajJXQN9zodb/NRW1jFLMZBXuK0mH16HBFN9GqCHIQDirYFzpCx6gw
zjOaqf7ntNf2f1XAN8hl5/Y3S6FkZaUyZyN1kITPIAQ4ww6r9ROD/RiYdHv7gYew
IV7EAAVRqSQABkRYlGrOcboRKMngeYVGHFXlJqxsbpcGztbjjf/pcRxk3Y6uhb2S
mVPb32yvAgMBAAECggEBAIb57viFMeLqFQ/KbkwjmHP+cshw8+LESSSUQgRa1vnw
v0G8lTFlqWGWlgHCcUNIBsYJ7ko0WAIFNv2ap2KjKVSqeUYnNLJ1lWn0t315UHnp
/1aomQTz/JBQ9TubbMHh8eK3KFUiYYhsaQRRuZ8sMQlQcilbXxF3fl2cDPem4gzp
ooXwuCW7GppKxpwOmap3Fy+p+EPUJ9IdBsu84rREDhlglv+8ASnYpVr9dZiAbK/F
iLreyJIIwK8rWLDRcik5UMlwuGFBwlijnRUEzi7ANE4sHcD/uWJutYV+9krzxjDM
vFe6Do464ma1MmMnPi80wptKkoKarjua3cLGSJdroDECgYEA3Ic3Mu90BbDkF07R
S6Bt3Kob0KiBpVNGdLNqf6Z4CpaCeLsLv4+zXJFZcA2DQmha1MuQGmgTJcF3+8IO
NU3ks4RV8llMyuQHkvuK7aqj123EMm7/H7mY+KEeC7Lyi0yZmKVRkakRj3XmSqQu
MlSPbT94jShKa7/P9uM+Q51Vq2kCgYEAy5B8MAWqL9j6F34vzuGgIyEi34XpAPEf
1Kw8o8OvnuFRjMRe2fb1n9/jyIwc0gUW7NbLFPbZaPCEOxbjR2LtpdmS6XCt/TZS
SY2t8ojy2c2qFgifxEjfOFKqQPhij+842uEJlNbgviMBVneZPfK+4nsNnbLxvL00
XbGNin0HTFcCgYA+ZyDOkAXDyn9wvQPqo5YS+Cvwyo4NK1hnk5GSV5fmXxrCcSNs
7IvzqMmnNJutAfyZ9JRtdH/ekjWSjyIYIVeTGOJ9NpnNW+NsyzNP95ZvUodPQit9
XbaUvHrVEqkhk+Zu1HEVh8MJVnJ5MqZD5bvETU6emwUcImYF1d37ohzo6QKBgGa7
9aD6yug49gazPYeIYRw5lfL/DxfVmT3o6vWvRcvGZTTIyiHwvAfCo5/L7qOjw+0l
ffqHljOa5vE3XN7jM5K3GqjLoFOhfaf3Y+l6ai232PYjxhX2vQkc1yXQ9VU04xm7
5u0CAQyUeBFebK1R/Doq5jVHYS7iwjHi8M8KyIsjAoGACXeMFLFYoJLb/EBDD9jl
JJ29G7Sn6c6UWqLsqUGIpt5n0G7PuM4twPOq/FIegKFnqDlTMdfGpRnoC76hgZ7e
nVl0vd8GzCtTE75E56YGUaAZtTFC8lF7i0FiCrXauwosknB38qFzONAbTx4JcMEP
Fl7qybzjFllYvka3aP4ae/M=
-----END PRIVATE KEY-----`

)

// TempCertFile creates a temp file with the contents set to proxyCertPEM
// and returns the temp file path.
func TempCertFile() (string, error) {
	f, err := ioutil.TempFile("", "zenoss_cert.")
	if err != nil {
		return "", err
	}
	defer f.Close()
	fmt.Fprint(f, InsecureCertPEM)
	return f.Name(), nil
}

// TempKeyFile creates a temp file with the contents set to proxyCertPEM
// and returns the temp file path.
func TempKeyFile() (string, error) {
	f, err := ioutil.TempFile("", "zenoss_key.")
	if err != nil {
		return "", err
	}
	defer f.Close()
	fmt.Fprint(f, InsecureKeyPEM)
	return f.Name(), nil
}
