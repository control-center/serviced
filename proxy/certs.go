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
	InsecureCertPEM = `-----BEGIN CERTIFICATE-----
MIICaDCCAdGgAwIBAgIJAMsgJclpgZqTMA0GCSqGSIb3DQEBBQUAME0xCzAJBgNV
BAYTAlVTMQ4wDAYDVQQIDAVUZXhhczEPMA0GA1UEBwwGQXVzdGluMQ8wDQYDVQQK
DAZaZW5vc3MxDDAKBgNVBAsMA0RldjAeFw0xMzA4MzAyMTE0MTBaFw0yMzA4Mjgy
MTE0MTBaME0xCzAJBgNVBAYTAlVTMQ4wDAYDVQQIDAVUZXhhczEPMA0GA1UEBwwG
QXVzdGluMQ8wDQYDVQQKDAZaZW5vc3MxDDAKBgNVBAsMA0RldjCBnzANBgkqhkiG
9w0BAQEFAAOBjQAwgYkCgYEAyY8M1eXgU+QJYyg/X3zKOfZf2NKOC1PEFzCJ9EUz
0tMkArHKCm3yid7Y2Jci2BMGlPKSgbp3wTGc32ONtSYxBOx7musmqgmD1LIADToL
UGaXPiolmMpv+GstaMFqpkWYfNCtlnzcTquMN+1jfKOd8+Ultodu4bZL4CJygfai
KRUCAwEAAaNQME4wHQYDVR0OBBYEFEes0lhAiq/5hAh01VxmE/eqqo2QMB8GA1Ud
IwQYMBaAFEes0lhAiq/5hAh01VxmE/eqqo2QMAwGA1UdEwQFMAMBAf8wDQYJKoZI
hvcNAQEFBQADgYEASJhY7kmME5Xv3k58C5IJXuVDCTJ1O/liHTCqzk3/GTvdvfKg
NiSsD6AUC/PVunaTs6ivwEFXcz7HFd94jsLfnEbfQ+tsTzct72vLknORxsuwAxpL
hXBOYfF12lYGYNlRN1HKFLSXysyHwCcWtGz886EUwzUWeCKOm7YGHYHUBaY=
-----END CERTIFICATE-----`

	InsecureKeyPEM = `-----BEGIN PRIVATE KEY-----
MIICeAIBADANBgkqhkiG9w0BAQEFAASCAmIwggJeAgEAAoGBAMmPDNXl4FPkCWMo
P198yjn2X9jSjgtTxBcwifRFM9LTJAKxygpt8one2NiXItgTBpTykoG6d8ExnN9j
jbUmMQTse5rrJqoJg9SyAA06C1Bmlz4qJZjKb/hrLWjBaqZFmHzQrZZ83E6rjDft
Y3yjnfPlJbaHbuG2S+AicoH2oikVAgMBAAECgYEAoQVK98aBhAN1DGYm2p3S4KNW
xtzO5XWx/eSlESQH1rEe35gxFEvpqwMAsWdsSrpIU83GBSV2bjy4Wi4qE0HDfgJ2
m3/IKGISTRyUZrnXprj1eIpwHbR5lhcKebohvtZeALKFH/8xdun0YzMkfJ4B2kxQ
7j28BBjKaowrBGOUeoECQQD83VkkJRCIVxxuSp+pvCJE2P9g6+j9NXlVcz3C0W0d
jJMIeVBQnqjy6bqpWqYwPCcroZ34Krc/o7OZAri2l+HdAkEAzA7YWIrDDih6x0BL
y4A/3kkGPj119u40woXicw1HMuW4X/zzXGfxHynO7KYqrTREKEJtBPUuGPC4JtXH
z0gcmQJAVyITEIBxJPoXgu3V/NAmYuD/hy9jlrUxfT97vcEav37sP5RGF7HEeAgQ
WUEyWRaxTLihTZ2yjYxkW8pzSgAmRQJBAJz5QoaCYGCQ1TpYBLaMdxVZWZshjpCh
WCbX9YaKDV5jBz2YCeHo970AXXUAss3A6jmKN/FbZtW6v/7n76hOEekCQQCV3ZhU
lhu+Iu4HUZGgpDg6tgnlB5Tv7zuyUlzPXgbNAsIsTvQfnmWa1/WpOvNOy2Ix5aJB
sl9SYPJBOM7G8o1p
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
