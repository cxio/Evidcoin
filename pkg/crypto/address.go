package crypto

import (
	"bytes"
	"crypto/sha256"
	"errors"
	"sort"

	"github.com/cxio/evidcoin/pkg/types"
	"github.com/mr-tron/base58"
	"golang.org/x/crypto/blake2b"
	"golang.org/x/crypto/sha3"
)

// Network identifies the address network and selects the text prefix (DEC-0104).
type Network string

const (
	// Mainnet uses the "Cx" address prefix.
	Mainnet Network = "Cx"
	// Testnet uses the "Tx" address prefix.
	Testnet Network = "Tx"
	// Devnet uses the "Dx" address prefix.
	Devnet Network = "Dx"
)

// Address errors.
var (
	// ErrMultisigRatio is returned for an invalid m/n ratio (m or n is 0, or m > n).
	ErrMultisigRatio = errors.New("crypto: invalid multisig m/n ratio")
	// ErrMultisigDuplicate is returned when multisig public keys contain a duplicate.
	ErrMultisigDuplicate = errors.New("crypto: duplicate multisig public key")
	// ErrUnknownNetwork is returned for an unrecognised network prefix.
	ErrUnknownNetwork = errors.New("crypto: unknown network prefix")
	// ErrBadChecksum is returned when an address checksum does not verify.
	ErrBadChecksum = errors.New("crypto: address checksum mismatch")
	// ErrBadAddress is returned for a malformed address (bad base58 or length).
	ErrBadAddress = errors.New("crypto: malformed address")
)

// AddressHashSingle derives a single-signature public key hash:
// SHA3-256( DomainTag("address.single") || BLAKE2b-512(pubKey) ) (DEC-0002/DEC-0104).
func AddressHashSingle(pubKey []byte) types.AddressHash {
	inner := blake2b.Sum512(pubKey)
	pre := make([]byte, 0, len(tagAddressSingle)+len(inner))
	pre = append(pre, tagAddressSingle...)
	pre = append(pre, inner[:]...)
	out := sha3.Sum256(pre)
	h, _ := types.NewAddressHash(out[:])
	return h
}

// AddressHashMulti derives a composite (multisig) public key hash for an
// m-of-n scheme (DEC-0104):
//  1. BaseH_i = BLAKE3-256(pubKey_i)
//  2. sort BaseH lexicographically and concatenate
//  3. PKHmix = SHA3-256( DomainTag("address.multi") || m || n || BaseHAll )
//
// m and n must be non-zero with m <= n, len(pubKeys) == n, and no duplicates.
func AddressHashMulti(m, n uint8, pubKeys [][]byte) (types.AddressHash, error) {
	var zero types.AddressHash
	if m == 0 || n == 0 || m > n {
		return zero, ErrMultisigRatio
	}
	if len(pubKeys) != int(n) {
		return zero, ErrMultisigRatio
	}

	baseHashes := make([][]byte, len(pubKeys))
	for i, pk := range pubKeys {
		bh := blake3_256(pk)
		baseHashes[i] = bh[:]
	}
	sort.Slice(baseHashes, func(i, j int) bool {
		return bytes.Compare(baseHashes[i], baseHashes[j]) < 0
	})
	// Reject duplicate public keys (their base hashes are now adjacent).
	for i := 1; i < len(baseHashes); i++ {
		if bytes.Equal(baseHashes[i-1], baseHashes[i]) {
			return zero, ErrMultisigDuplicate
		}
	}

	pre := make([]byte, 0, len(tagAddressMulti)+2+32*len(baseHashes))
	pre = append(pre, tagAddressMulti...)
	pre = append(pre, m, n)
	for _, bh := range baseHashes {
		pre = append(pre, bh...)
	}
	inner := blake2b.Sum512(pre)
	out := sha3.Sum256(inner[:])
	return types.NewAddressHash(out[:])
}

// EncodeAddress renders a 32-byte public key hash as address text:
// prefix || Base58(pubKeyHash || checksum), where
// checksum = last4(SHA2-256(SHA2-256(prefix || pubKeyHash))) (DEC-0104).
// The prefix participates in the checksum but is not part of the base58 payload.
func EncodeAddress(net Network, h types.AddressHash) (string, error) {
	prefix := string(net)
	if !validNetwork(net) {
		return "", ErrUnknownNetwork
	}
	hash := h.Bytes()
	cksum := addressChecksum(prefix, hash)
	payload := make([]byte, 0, len(hash)+4)
	payload = append(payload, hash...)
	payload = append(payload, cksum...)
	return prefix + base58.Encode(payload), nil
}

// DecodeAddress recovers the network and 32-byte public key hash from address
// text, verifying the checksum. It rejects unknown prefixes, invalid base58,
// wrong length and checksum failures.
func DecodeAddress(addr string) (Network, types.AddressHash, error) {
	var zero types.AddressHash
	if len(addr) < 2 {
		return "", zero, ErrBadAddress
	}
	net := Network(addr[:2])
	if !validNetwork(net) {
		return "", zero, ErrUnknownNetwork
	}
	payload, err := base58.Decode(addr[2:])
	if err != nil {
		return "", zero, ErrBadAddress
	}
	if len(payload) != 32+4 {
		return "", zero, ErrBadAddress
	}
	hash := payload[:32]
	gotSum := payload[32:]
	wantSum := addressChecksum(string(net), hash)
	if !bytes.Equal(gotSum, wantSum) {
		return "", zero, ErrBadChecksum
	}
	h, err := types.NewAddressHash(hash)
	if err != nil {
		return "", zero, err
	}
	return net, h, nil
}

func validNetwork(net Network) bool {
	switch net {
	case Mainnet, Testnet, Devnet:
		return true
	default:
		return false
	}
}

// addressChecksum returns last4(SHA2-256(SHA2-256(prefix || pubKeyHash))).
func addressChecksum(prefix string, pubKeyHash []byte) []byte {
	buf := make([]byte, 0, len(prefix)+len(pubKeyHash))
	buf = append(buf, prefix...)
	buf = append(buf, pubKeyHash...)
	first := sha256.Sum256(buf)
	second := sha256.Sum256(first[:])
	out := make([]byte, 4)
	copy(out, second[28:])
	return out
}
