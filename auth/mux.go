package auth

import (
	"bytes"
	"encoding/binary"
	"errors"
)

/*
   When establishing a connection to the mux, in addition to the address of the receiver,
   the sender sends an authentication token and signs the whole message. The token determines
   if the sender is authorized to send data to the receiver or not

   ----------------------------------------------------------------------------------------
   | address (6 bytes) |  token length (4 bytes)  |   Auth Token (N bytes)  |  Signature  |
   ----------------------------------------------------------------------------------------
*/

var (
	endian           = binary.BigEndian
	ErrBadMuxAddress = errors.New("Bad mux address")
	ErrBadMuxHeader  = errors.New("Bad mux header")
)

const (
	ADDRESS_BYTES   = 6
	TOKEN_LEN_BYTES = 4
)

func BuildMuxHeader(address []byte) ([]byte, error) {
	headerBuf := new(bytes.Buffer)

	//get current host token
	token := AuthToken()

	// add address
	if len(address) != ADDRESS_BYTES {
		return nil, ErrBadMuxAddress
	}
	headerBuf.Write([]byte(address))

	// add token length
	var tokenLen uint32 = uint32(len(token))
	tokenLenBuf := make([]byte, 4)
	endian.PutUint32(tokenLenBuf, tokenLen)
	headerBuf.Write(tokenLenBuf)

	// add token
	headerBuf.Write([]byte(token))

	// Sign what we have so far
	myPrivateKey := LocalPrivateKey()
	signer, err := RSASigner(myPrivateKey)
	if err != nil {
		return nil, err
	}
	signature, err := signer.Sign(headerBuf.Bytes())
	if err != nil {
		return nil, err
	}
	// add signature to header
	headerBuf.Write(signature)

	return headerBuf.Bytes(), nil
}

func errorExtractingHeader(err error) (string, Identity, error) {
	return "", nil, err
}

func ExtractMuxHeader(rawHeader []byte) (string, Identity, error) {
	var offset uint32 = 0

	if len(rawHeader) < ADDRESS_BYTES+TOKEN_LEN_BYTES {
		return errorExtractingHeader(ErrBadMuxHeader)
	}

	// First six bytes is going to be the address
	address := string(rawHeader[offset:ADDRESS_BYTES])
	offset += ADDRESS_BYTES

	// Next four bytes represents the token length
	tokenLen := endian.Uint32(rawHeader[offset : offset+TOKEN_LEN_BYTES])
	offset += TOKEN_LEN_BYTES
	if len(rawHeader) <= ADDRESS_BYTES+TOKEN_LEN_BYTES+int(tokenLen) {
		return errorExtractingHeader(ErrBadMuxHeader)
	}

	// Next tokeLen bytes for the token
	token := string(rawHeader[offset : offset+tokenLen])
	offset += tokenLen

	// whole message that has been signed
	signed_message := rawHeader[:offset]

	// Whatever is left is the signature
	signature := rawHeader[offset:]

	// Extract token
	masterPublicKey := MasterPublicKey()
	senderIdentity, err := ParseJWTIdentity(token, &masterPublicKey)
	if err != nil {
		return errorExtractingHeader(err)
	}

	// Verify the signed message with the sender's public key
	senderPublicKey := senderIdentity.PublicKey()
	senderVerifier, err := RSAVerifier(senderPublicKey)
	if err != nil {
		return errorExtractingHeader(err)
	} else {
		err := senderVerifier.Verify(signed_message, signature)
		if err != nil {
			return errorExtractingHeader(err)
		}
	}

	return address, senderIdentity, nil
}
