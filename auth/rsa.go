package auth

import (
	"bytes"
	"crypto"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/pem"
)

const rsaKeyLength = 2048

var (
	// DevPubKeyPEM is a sample public key for use in dev
	DevPubKeyPEM = []byte(`-----BEGIN PUBLIC KEY-----
MIIBIjANBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAxeGhO/4jJ7fPwXHjtZx+
q/Ne+fhMEzGB41aD6QKij6u0LPBWynmXdJeLdIW1N8ZFF7PdpA4qAu6ouMRvOuSJ
1qPt1hToahBxxducEp64nQ/fWN0uANjPqjlKcjj/fiSZ2ewrXYAOmnbaIQgt3fjv
VYQgdGmHA5uyROclsutOF0shyprU2x/S8uXIK1fJM/yxukcDG6GvymW0b5mqLZZA
Zmpt11QJ8YV5yiBtziSyYfiXTFs5yoydvRqmTIRm1CBnV3JYXio9fXv4C1BVTk11
miqYybTUZga1O9mykjDbrwtaigb2rP1EjQzJoMLHW27edXBZUFQjedD0N20+WkUx
0wIDAQAB
-----END PUBLIC KEY-----`)

	// DevPrivKeyPEM is a sample private key for use in dev
	DevPrivKeyPEM = []byte(`-----BEGIN RSA PRIVATE KEY-----
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

func parsePEM(key []byte) ([]byte, error) {
	var block *pem.Block
	if block, _ = pem.Decode(key); block == nil {
		return nil, ErrNotPEMEncoded
	}
	return block.Bytes, nil
}

type rsaVerifier struct {
	pubkey *rsa.PublicKey
}

func (v *rsaVerifier) Verify(message []byte, signature []byte) error {
	hashed := sha256.Sum256(message)
	return rsa.VerifyPSS(v.pubkey, crypto.SHA256, hashed[:], signature, nil)
}

type rsaSigner struct {
	privkey *rsa.PrivateKey
}

func (s *rsaSigner) Sign(message []byte) ([]byte, error) {
	hashed := sha256.Sum256(message)
	return rsa.SignPSS(rand.Reader, s.privkey, crypto.SHA256, hashed[:], nil)
}

// RSAPrivateKeyFromPEM decodes a PEM-encoded RSA private key
func RSAPrivateKeyFromPEM(key []byte) (*rsa.PrivateKey, error) {
	parsed, err := parsePEM(key)
	if err != nil {
		return nil, err
	}
	var parsedKey crypto.PrivateKey
	if parsedKey, err = x509.ParsePKCS1PrivateKey(parsed); err != nil {
		if parsedKey, err = x509.ParsePKCS8PrivateKey(parsed); err != nil {
			return nil, ErrNotRSAPrivateKey
		}
	}
	return verifyRSAPrivateKey(parsedKey)
}

// RSAPublicKeyFromPEM decodes a PEM-encoded RSA public key
func RSAPublicKeyFromPEM(key []byte) (*rsa.PublicKey, error) {
	parsed, err := parsePEM(key)
	if err != nil {
		return nil, err
	}
	var parsedKey crypto.PublicKey
	if parsedKey, err = x509.ParsePKIXPublicKey(parsed); err != nil {
		cert, err := x509.ParseCertificate(parsed)
		if err != nil {
			return nil, ErrNotRSAPublicKey
		}
		parsedKey = cert.PublicKey
	}
	return verifyRSAPublicKey(parsedKey)
}

func verifyRSAPrivateKey(key crypto.PrivateKey) (*rsa.PrivateKey, error) {
	pkey, ok := key.(*rsa.PrivateKey)
	if !ok {
		return nil, ErrNotRSAPrivateKey
	}
	return pkey, nil
}

func verifyRSAPublicKey(key crypto.PublicKey) (*rsa.PublicKey, error) {
	pkey, ok := key.(*rsa.PublicKey)
	if !ok {
		return nil, ErrNotRSAPublicKey
	}
	return pkey, nil
}

// RSASigner creates a Signer from a key, validating that it is an RSA private
// key first
func RSASigner(key crypto.PrivateKey) (Signer, error) {
	pkey, err := verifyRSAPrivateKey(key)
	if err != nil {
		return nil, err
	}
	return &rsaSigner{pkey}, nil
}

// RSASignerFromPEM creates a Signer from a PEM-encoded key, validating that it
// is an RSA private key first
func RSASignerFromPEM(key []byte) (Signer, error) {
	pkey, err := RSAPrivateKeyFromPEM(key)
	if err != nil {
		return nil, err
	}
	return RSASigner(pkey)
}

// RSAVerifier creates a Verifier from a key, verifying that it is an RSA
// public key first
func RSAVerifier(key crypto.PublicKey) (Verifier, error) {
	pkey, err := verifyRSAPublicKey(key)
	if err != nil {
		return nil, err
	}
	return &rsaVerifier{pkey}, nil
}

// RSAVerifierFromPEM creates a Verifier from a PEM-encoded key, verifying that
// it is an RSA public key first
func RSAVerifierFromPEM(key []byte) (Verifier, error) {
	pkey, err := RSAPublicKeyFromPEM(key)
	if err != nil {
		return nil, err
	}
	return RSAVerifier(pkey)
}

// PEMFromRSAPublicKey creates a PEM block from an RSA public key
func PEMFromRSAPublicKey(key crypto.PublicKey, headers map[string]string) ([]byte, error) {
	pkey, err := verifyRSAPublicKey(key)
	if err != nil {
		return nil, err
	}
	marshalled, err := x509.MarshalPKIXPublicKey(pkey)
	if err != nil {
		return []byte{}, err
	}
	block := pem.Block{
		Type:    "PUBLIC KEY",
		Headers: headers,
		Bytes:   marshalled,
	}
	bytes := pem.EncodeToMemory(&block)
	return bytes, nil
}

// PEMFromRSAPrivateKey creates a PEM block from an RSA public key
func PEMFromRSAPrivateKey(key crypto.PrivateKey, headers map[string]string) ([]byte, error) {
	pkey, err := verifyRSAPrivateKey(key)
	if err != nil {
		return nil, err
	}
	marshalled := x509.MarshalPKCS1PrivateKey(pkey)
	block := pem.Block{
		Type:    "RSA PRIVATE KEY",
		Headers: headers,
		Bytes:   marshalled,
	}
	bytes := pem.EncodeToMemory(&block)
	return bytes, nil
}

// GenerateKey generates an RSA key pair and returns the public and private
// PEM blocks for that key.
func GenerateRSAKeyPairPEM(headers map[string]string) (public []byte, private []byte, err error) {
	privateKey, err := rsa.GenerateKey(rand.Reader, rsaKeyLength)
	if err != nil {
		return nil, nil, err
	}
	if private, err = PEMFromRSAPrivateKey(privateKey, headers); err != nil {
		return nil, nil, err
	}
	publicKey := privateKey.Public()
	if public, err = PEMFromRSAPublicKey(publicKey, headers); err != nil {
		return nil, nil, err
	}
	return public, private, nil
}

// DevRSASigner returns a dev signer for dev purposes
func DevRSASigner() Signer {
	signer, _ := RSASignerFromPEM(DevPrivKeyPEM)
	return signer
}

// DevRSAVerifier returns a dev verifier for dev purposes
func DevRSAVerifier() Verifier {
	verifier, _ := RSAVerifierFromPEM(DevPubKeyPEM)
	return verifier
}

// TODO: Elimnate these three methods.  Leaving these here for now so the code will build
func LocalPrivateKey() crypto.PrivateKey {
	return delegateKeys.localPrivate
}

func LocalPublicKey() crypto.PublicKey {
	key, _ := RSAPublicKeyFromPEM(DevPubKeyPEM)
	return key
}

func MasterPublicKey() crypto.PublicKey {
	masterPublic, _ := GetMasterPublicKey()
	return masterPublic
}

// DumpPEMKeyPair dumps PEM-encoded public and private keys to a single byte array
func DumpPEMKeyPair(public, private []byte) ([]byte, error) {
	// Do some validation first.  These don't have to be a matched pair,
	//   but they do have to be PEM, and they have to be a public and
	//	 private key, respectively
	if _, err := RSAPublicKeyFromPEM(public); err != nil {
		return nil, err
	}

	if _, err := RSAPrivateKeyFromPEM(private); err != nil {
		return nil, err
	}

	var out bytes.Buffer
	if _, err := out.Write(private); err != nil {
		return nil, err
	}
	if _, err := out.Write(public); err != nil {
		return nil, err
	}

	return out.Bytes(), nil
}

// LoadPEMKeyPair loads a private/public key pair from a reader over PEM-encoded data.
//  The private key is first, the public key is second.
func LoadKeyPair(data []byte) (public crypto.PublicKey, private crypto.PrivateKey, err error) {
	firstblock, rest := pem.Decode(data)
	if firstblock == nil {
		return nil, nil, ErrBadKeysFile
	}
	privatekey, err := RSAPrivateKeyFromPEM(pem.EncodeToMemory(firstblock))
	if err != nil {
		return nil, nil, err
	}
	secondblock, _ := pem.Decode(rest)
	if secondblock == nil {
		return nil, nil, ErrBadKeysFile
	}
	publickey, err := RSAPublicKeyFromPEM(pem.EncodeToMemory(secondblock))
	if err != nil {
		return nil, nil, err
	}
	return publickey, privatekey, nil
}
