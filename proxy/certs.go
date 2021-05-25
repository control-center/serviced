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
MIID/TCCAuWgAwIBAgIUCHAZe7nuNsnmGqoE7g7+agm6NjQwDQYJKoZIhvcNAQEL
BQAwgY0xCzAJBgNVBAYTAlVTMQ4wDAYDVQQIDAVUZXhhczEPMA0GA1UEBwwGQXVz
dGluMQ8wDQYDVQQKDAZaZW5vc3MxFjAUBgNVBAsMDUNvbnRyb2xDZW50ZXIxFjAU
BgNVBAMMDUNvbnRyb2xDZW50ZXIxHDAaBgkqhkiG9w0BCQEWDWl0QHplbm9zcy5j
b20wHhcNMjEwMzAzMTIxNjE5WhcNMjYwMzAzMTIxNjE5WjCBjTELMAkGA1UEBhMC
VVMxDjAMBgNVBAgMBVRleGFzMQ8wDQYDVQQHDAZBdXN0aW4xDzANBgNVBAoMBlpl
bm9zczEWMBQGA1UECwwNQ29udHJvbENlbnRlcjEWMBQGA1UEAwwNQ29udHJvbENl
bnRlcjEcMBoGCSqGSIb3DQEJARYNaXRAemVub3NzLmNvbTCCASIwDQYJKoZIhvcN
AQEBBQADggEPADCCAQoCggEBAMGxbd835EsuHa9rSYKPFeb89R9uOS3GcOR32da6
K50HSsWdM0vbA6oIzkOaa7aO7o085kjsVOai1IQujCkOeMeGYXQsoz/8+J8RKLTx
7aG/wo0S+OWyeNBs9hJlcBH6wPFhM86PN1aqdtZe/dvZ5fA4iuU87ic7HwEoC/SZ
X/j4zxAcDOD3K5mj468WWbrpR5hrngmAZWalhGCBzBmC0qVYYZhzmzeILDz/Cf+j
0hx1jq+120HZ5CFDR0tXdoBxFc2PkNqvooMj4YJ245hQPwP8bss+kyOo0pSoU2zF
4hS0tkbUTJPmIa4iyaPbyOwp+yQ44gn7DOYbEmrNxRXLiUUCAwEAAaNTMFEwHQYD
VR0OBBYEFGpX0Y3yztbQ149snu2y5qKGbJCTMB8GA1UdIwQYMBaAFGpX0Y3yztbQ
149snu2y5qKGbJCTMA8GA1UdEwEB/wQFMAMBAf8wDQYJKoZIhvcNAQELBQADggEB
AIjFipVyLP8k5kN1xgIAWJYsC9QKfBYt+rrZInU4X+dFI+cmSLgs7XCv6nvPxAWw
lu4bNG8mGMlUtmiVvXKhR+W1PBySabL1sHLFNKAahb2oJPcbR9K2nPJULWo2hLfc
ul7z1Juk6ZmjW57OaPNQYQd29ZkkgrtjPFXmDRLZc7HMb3SuLOku8IBwSdYZugSW
AEZOzAlzmpaCt9wFc50+o5S4o66Cotz23JyFbjMXQzdoGCO28JMxm2/vELbjzmDz
IiZk2Q4bLlkByheK6S/XPl2gR7J3bBdLBqewhrysugJlO6FS0MYx8GS3L7SPcPjo
egF8qBbgik/f/cwI9mwCdlU=
-----END CERTIFICATE-----`

	InsecureKeyPEM = `-----BEGIN PRIVATE KEY-----
MIIEvQIBADANBgkqhkiG9w0BAQEFAASCBKcwggSjAgEAAoIBAQDBsW3fN+RLLh2v
a0mCjxXm/PUfbjktxnDkd9nWuiudB0rFnTNL2wOqCM5Dmmu2ju6NPOZI7FTmotSE
LowpDnjHhmF0LKM//PifESi08e2hv8KNEvjlsnjQbPYSZXAR+sDxYTPOjzdWqnbW
Xv3b2eXwOIrlPO4nOx8BKAv0mV/4+M8QHAzg9yuZo+OvFlm66UeYa54JgGVmpYRg
gcwZgtKlWGGYc5s3iCw8/wn/o9IcdY6vtdtB2eQhQ0dLV3aAcRXNj5Dar6KDI+GC
duOYUD8D/G7LPpMjqNKUqFNsxeIUtLZG1EyT5iGuIsmj28jsKfskOOIJ+wzmGxJq
zcUVy4lFAgMBAAECggEAXOZFxV8gTvKyaDV0D3ujTKOcydNq90qLUPku8S9fxbP+
p+Nq/AHysvYAQCpraScKwZEo/mbjna8RcMiGjxaS1VfdnAPg1Mr3UAvB02+Jwx+f
J1ynJjxAd+8a4t3mL6luKxes6nCEYTvnPZBX/7916o6kB6j+rLBNgZd8jHeXsWiF
v9ypSZZeWsUKrid2tQQpCjY0jibAp4KGl8wwGDGogrR6spHuWVFITnuKb42APD6q
nKdOsrnYOZ3YPcuYJ0j08BH/7cpD4kcaVVFfbKd/ujUwJXUy3LdL9zBWhP+fBd87
s/+PkoHhbu89YFyT5iHHIbKN4ugft3nWqe5Ei2yz7QKBgQD4Tn+8ZbmRS8An5DzP
6B/kAjiuxdY4VA5pZ5RidxRyK1jZOEU1yS/4C8beL1Ykn56+hVyExV2hD6cFJUdY
Sy6k49O5VWlIjLmit8Xo5lfgVdWhbXQPu39KPZTckiqFtJ4GouGAG62XLrAXLssi
0sW7g4DoQuHBYiXGCaxYWvXS/wKBgQDHscATJxrHNfmEE1lloE0bPvaafta6HRQi
y+INXVIR1QkKv9bSNmeAf4lMGNIjb4H3iEtRkRxKNSS3qF107frLNW+DcMQPux3O
VO1SknG3h2bSXJlkc3kvvg7uJiTo7P/UF6syjL+8qxRqg4NuvI6pdY5vWnQjwASX
nDnpKTqXuwKBgQDzTD9fAzGji1y5+aoYcTKmQAL4RQMU6E/Cueor3NAc2hpRpRAz
lnE5E5kFZc57TifGOHgh5B1MzkByC0fv3KLUkCOJqoXhv3m5VWZHQUQDnTcY2F1r
eOVNgi+VPGcL4aEhkYFw/C3IP8fsvz3tXia/CChL7BS2XovyktHbNS0/UwKBgDR9
Cil+m9FE5KLMmzDVI69Hq3YMZNBimEpVIMO2hb3eKxRCPGrgle/2ldYEqCdcReMU
VgfIhpESyuXjQT1c2BDVqMv5te8Ulc8ID6EmkPFWi7Y7VK5Mk8vyvuXl7Mm0kcHj
vsH4sOUcaq9chg1zTmRIW/n04pYLAKoBDE+24InFAoGAd7bzk/HAOTSY3eynp0Bd
TKuuaMu9nPs9k6soWCqPiaFS4bo2IfyH7wo50a8HhXiZA/knJJOJ+j3K3SQiI/0H
PYsguL6jJUs5lXrcznu+3H0REVrjh5w2zWGO+bGl3mtMLvBEVKepQZ7Kkf+Y1dH7
vWz/x0dRiDVmHkdlD3xlkwQ=
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
