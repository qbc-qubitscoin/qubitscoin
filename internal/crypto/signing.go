package crypto

import (
	"errors"

	"github.com/cloudflare/circl/sign/mldsa/mldsa65"
)

var scheme = mldsa65.Scheme()

// Sign signs msg with the given ML-DSA-65 private key bytes.
func Sign(privKeyBytes, msg []byte) ([]byte, error) {
	priv := new(mldsa65.PrivateKey)
	if err := priv.UnmarshalBinary(privKeyBytes); err != nil {
		return nil, err
	}
	sig := scheme.Sign(priv, msg, nil)
	return sig, nil
}

// Verify checks an ML-DSA-65 signature and returns false (not an error) for
// invalid signatures, reserving errors for key-parsing failures.
func Verify(pubKeyBytes, msg, sig []byte) (bool, error) {
	pub := new(mldsa65.PublicKey)
	if err := pub.UnmarshalBinary(pubKeyBytes); err != nil {
		return false, err
	}
	if len(sig) != SignatureSize {
		return false, errors.New("invalid signature length")
	}
	return scheme.Verify(pub, msg, sig, nil), nil
}
