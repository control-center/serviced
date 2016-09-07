package auth

import (
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
)

var (
	pubkey  *rsa.PublicKey
	privkey *rsa.PrivateKey
)

func init() {
	var (
		pub = []byte(`-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxeGhO/4jJ7fPwXHjtZx+
q/Ne+fhMEzGB41aD6QKij6u0LPBWynmXdJeLdIW1N8ZFF7PdpA4qAu6ouMRvOuSJ
1qPt1hToahBxxducEp64nQ/fWN0uANjPqjlKcjj/fiSZ2ewrXYAOmnbaIQgt3fjv
VYQgdGmHA5uyROclsutOF0shyprU2x/S8uXIK1fJM/yxukcDG6GvymW0b5mqLZZA
Zmpt11QJ8YV5yiBtziSyYfiXTFs5yoydvRqmTIRm1CBnV3JYXio9fXv4C1BVTk11
miqYybTUZga1O9mykjDbrwtaigb2rP1EjQzJoMLHW27edXBZUFQjedD0N20+WkUx
0wIDAQAB
-----END PUBLIC KEY-----`)

		priv = []byte(`-----BEGIN RSA PRIVATE KEY-----
MIIEpQIBAAKCAQEAxeGhO/4jJ7fPwXHjtZx+q/Ne+fhMEzGB41aD6QKij6u0LPBW
ynmXdJeLdIW1N8ZFF7PdpA4qAu6ouMRvOuSJ1qPt1hToahBxxducEp64nQ/fWN0u
ANjPqjlKcjj/fiSZ2ewrXYAOmnbaIQgt3fjvVYQgdGmHA5uyROclsutOF0shyprU
2x/S8uXIK1fJM/yxukcDG6GvymW0b5mqLZZAZmpt11QJ8YV5yiBtziSyYfiXTFs5
yoydvRqmTIRm1CBnV3JYXio9fXv4C1BVTk11miqYybTUZga1O9mykjDbrwtaigb2
rP1EjQzJoMLHW27edXBZUFQjedD0N20+WkUx0wIDAQABAoIBAQC5W2HZvXOWx9Jf
JNylCOOLFFx6EIDKVcySdD757BL0O2V51uOlpAIgH7hsvLaEeT/NNRm/i9kEiCQZ
dz+tzdfj7AhkSY9vktnP+aeXtX++99hB+GUYO/9CN4optoR017oZ7OSuH78OJ5ip
6mx0ibM+ypZQFW14DscNTev9TfcHTx2Id9/iQx1bZFXxWyhjpuAlxliw3hkS4ANt
axI9N8Wi+K7uHYkMsDul32JIHFPPgpD4uB8mcCsLG48sRPqC/wcqeb48Dift6c/y
wo3JWk8Q5wi8/pE+ryVccSHa5KHaO+bhVr/z5ItNgA9MQdE3k5N7Umx41rgQR0Xr
+CYwH2JhAoGBAPLGIi5TK/hb+0Z9NQvoqOOObju8U4gOv5n+diW+xMU4EoJBsllM
piKfrlXBn+09hovx72p2LWYXQERp7yfd0AzcWC41HqGjtQ+1gA+9GDYC/wLDA+pl
tW7tlmzcIAIsYPUFBSPSb+WA4aJygir/1Hta/PldYGGrhUxfhrzZbeXxAoGBANCp
Ze3pG7N2mDuSxEhtXOOD9JMupz3tMZr2p+QgtjR5MRWR9+G3mFna5462Rg9d/d7z
4adxsvn/7OE/9stkLpdmcZrq0jx5s9V/EQkliO7LG9l3H/JoPEYPt5alGpeV3awG
7uTrkpdqO44U2PTCPCZ7i07axeHC9J68GzDMM4ADAoGBAMNto45O/ZJL2RaBK/aO
L4Ye3bXQgB2CYdKA+HKiApwP6zZX1E32WbZ9fEUkPK0pXenBs8yrnRgVl3J7JD2f
XR89MO7ha+sKcXJX1OLWgWrZNpbujXRes5K8Rt8Sw+F8AAC9LcoMWG8TNI8kRox+
rHkwYXwLIs78160HKNtU3BbBAoGBAMgCCA0TC5VrUSKRXQnbolUG8BGAf5hxWsIi
Oe4GmQAVRsJZR1SZqjQ/CwQVnXQvcSAbfyoEZz0RXprOuB5fafV/odePzHNhaMp1
YPv2eZoDIC/D6uBtn5C8kgqZObMhWPkDMExHrhzrHCjlvMxnvkZY18B/HXx4Zggd
YKbWpWrHAoGAYhE/GcHnJ/SJznMl0O9jkg/I0TQwDRnibzvESb3M2aZZ7slL/AYb
x9eZ/rWmKDZN74BDBmCIVBn8jgXk0Qv9JmltOS7b6Md4R5DTaeP6QJEEBGpKkFI4
gTU6k22ENbaM2VIHhEjJQYftvA63316pfDqF31yq/cpspdaNrntc7xc=
-----END RSA PRIVATE KEY-----`)
	)
	privblock, _ := pem.Decode(priv)
	privparsed, _ := x509.ParsePKCS1PrivateKey(privblock.Bytes)
	privkey = privparsed

	pubblock, _ := pem.Decode(pub)
	pubparsed, _ := x509.ParsePKIXPublicKey(pubblock.Bytes)
	if typed, ok := pubparsed.(*rsa.PublicKey); ok {
		pubkey = typed
	}
}

type devKeyStore struct{}

func (k *devKeyStore) Sign(message []byte) ([]byte, error) {
	hashed := sha256.Sum256(message)
	return rsa.SignPSS(rand.Reader, privkey, crypto.SHA256, hashed[:], nil)
}

func (k *devKeyStore) Verify(message []byte, signature []byte) error {
	hashed := sha256.Sum256(message)
	return rsa.VerifyPSS(pubkey, crypto.SHA256, hashed[:], signature, nil)
}

// DevRSASigner returns a dev signer for dev purposes
func DevRSASigner() Signer {
	return &devKeyStore{}
}

// DevRSAVerifier returns a dev verifier for dev purposes
func DevRSAVerifier() Verifier {
	return &devKeyStore{}
}
