package validator

import (
	"errors"

	"github.com/tendermint/tendermint/crypto"
	ed "github.com/tendermint/tendermint/crypto/ed25519"
	secp "github.com/tendermint/tendermint/crypto/secp256k1"
)

func byteToPubKey(key []byte) (crypto.PubKey, error) {
	switch len(key) {
	case ed.PubKeySize:
		return ed.PubKey(key), nil
	case secp.PubKeySize:
		return secp.PubKey(key), nil
	default:
		return nil, errors.New("invalid len of public key")
	}
}

func pubKeyToByte(key crypto.PubKey) ([]byte, error) {
	switch v := key.(type) {
	case ed.PubKey:
		return []byte(v), nil
	case secp.PubKey:
		return []byte(v), nil
	default:
		return []byte{}, errors.New("invalid public key type")
	}
}
