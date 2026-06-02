package services

import (
	"github.com/cxio/evidcoin/internal/validation"
	"github.com/cxio/evidcoin/pkg/hashtree"
	"github.com/cxio/evidcoin/pkg/types"
)

// StateKind 标识状态证明条目的类型（UTXO 或 UTCO）。
type StateKind uint8

const (
	// StateKindUTXO 表示 UTXO（未花费币金输出）状态条目。
	StateKindUTXO StateKind = 1
	// StateKindUTCO 表示 UTCO（未转出凭信输出）状态条目。
	StateKindUTCO StateKind = 2
)

// TxLookupResponse is the response for a TxLookup query (第 15 章 §3，DEC-0603).
//
// It returns the complete transaction data for a given year and TxID,
// along with the block height and in-block sequence position.
// The returned TxData must be verifiable via TxID (recompute TxID from TxData
// and compare against the queried TxID).
type TxLookupResponse struct {
	// Year is the transaction year (UTC calendar year).
	Year uint64
	// TxID is the queried transaction ID (SHA3-384, 48 bytes).
	TxID types.TxID
	// TxData is the complete serialized transaction in canonical encoding.
	TxData []byte
	// BlockHeight is the block height at which this transaction was confirmed.
	BlockHeight uint32
	// TxIndex is the in-block sequence position (0-based, Coinbase at 0).
	TxIndex uint32
}

// TxProofResponse is the response for a TxProof query (第 15 章 §3，DEC-0603).
//
// It returns the Merkle proof path from the transaction leaf to the block's
// transaction tree root (see proposal §4, DEC-0004).
// The client must verify the proof locally: recompute the leaf hash from the TxID,
// walk the sibling chain, and compare the result to the known tree root.
type TxProofResponse struct {
	// TxID is the transaction whose Merkle proof is provided.
	TxID types.TxID
	// BlockHeight is the block that contains the transaction.
	BlockHeight uint32
	// Proof is the Merkle path from the transaction leaf hash to the tree root.
	Proof hashtree.Proof
}

// BlockTxListResponse is the response for a BlockTxList query (第 15 章 §3，DEC-0603).
//
// It returns either the complete TxID sequence for a block, or a network block summary.
// When IsSummary is false, TxIDs carries the full list and the client can recompute
// the transaction tree root for verification.
// When IsSummary is true, Summary carries the compact representation; final verification
// still requires obtaining the complete TxID sequence.
type BlockTxListResponse struct {
	// BlockID is the requested block identifier (SHA3-384, 48 bytes).
	BlockID types.BlockID
	// BlockHeight is the block height.
	BlockHeight uint32
	// TxIDs contains the full TxID sequence when IsSummary is false.
	// Ordered by in-block sequence position; Coinbase is at index 0.
	TxIDs []types.TxID
	// Summary carries the network block summary when IsSummary is true.
	// Nil when IsSummary is false.
	Summary *BlockSummary
	// IsSummary indicates whether this response carries a summary rather than full TxIDs.
	IsSummary bool
}

// StateProofEntry is a single UTXO or UTCO state proof entry (第 15 章 §3，DEC-0603).
//
// Each entry carries the state bit proof (Merkle path to the state root) and the
// output payload for reference. The client must verify Kind, TxID, OutIndex, and IsValid
// against the known UTXO/UTCO root (via CheckRoot).
type StateProofEntry struct {
	// Kind indicates whether this is a UTXO (StateKindUTXO=1) or UTCO (StateKindUTCO=2) entry.
	Kind StateKind
	// TxID is the originating transaction ID (SHA3-384, 48 bytes).
	TxID types.TxID
	// OutIndex is the output sequence index within the originating transaction.
	OutIndex uint64
	// IsValid indicates whether the output is currently unspent (valid in the state set).
	IsValid bool
	// Proof is the Merkle path from the state leaf to the UTXO or UTCO root.
	// The client must verify this path against the locally-held state root.
	Proof hashtree.Proof
	// OutputData is the raw canonical-encoded output payload, for reference and verification.
	OutputData []byte
}

// StateProofResponse is the response for a StateProof query (第 15 章 §3，DEC-0603).
//
// It returns UTXO/UTCO state bit proofs and output details (see proposal §9, DEC-0201).
// The client must verify each entry's proof against the locally-held UTXO/UTCO roots,
// which are themselves committed by the block's CheckRoot.
type StateProofResponse struct {
	// Entries is the list of state proof entries.
	Entries []StateProofEntry
	// UTXORoot is the UTXO state root against which these proofs were computed.
	// The client must confirm this matches the locally-known UTXO root.
	UTXORoot types.TreeHash
	// UTCORoot is the UTCO state root against which these proofs were computed.
	// The client must confirm this matches the locally-known UTCO root.
	UTCORoot types.TreeHash
}

// RecentBlockProofsResponse is the response for a RecentBlockProofs query
// (第 15 章 §3·§6，DEC-0601，DEC-0603).
//
// It returns at least MinRecentBlockProofs (31) consecutive block proof packages to
// cover the fork safety window. Initial node synchronization depends on the completeness
// of this response (see proposal §13).
// Use ValidateRecentBlockProofs to verify the minimum count requirement.
type RecentBlockProofsResponse struct {
	// ProofPackages is the list of block proof packages, ordered from oldest to newest.
	// Must contain at least MinRecentBlockProofs (31) entries.
	ProofPackages []validation.ProofPackage
}

// AttachmentIndexResponse is the response for an AttachmentIndex query
// (第 15 章 §3，DEC-0603).
//
// For small attachments (< DataBoundaryBytes), Data carries the raw attachment bytes.
// For large attachments (>= DataBoundaryBytes), FragmentIndex carries the serialized
// fragment index for retrieving the data via Depots.
// The client must verify the data against the known attachment fingerprint (SHA3-512).
type AttachmentIndexResponse struct {
	// Fingerprint is the canonical attachment fingerprint (SHA3-512, 64 bytes).
	Fingerprint types.AttachmentHash
	// IsLargeAttachment indicates whether this attachment reaches or exceeds DataBoundaryBytes.
	IsLargeAttachment bool
	// Data contains the raw attachment bytes for small attachments (< DataBoundaryBytes).
	// Empty when IsLargeAttachment is true.
	Data []byte
	// FragmentIndex contains the serialized fragment index for large attachments.
	// Empty when IsLargeAttachment is false.
	FragmentIndex []byte
	// FragmentCount is the number of fragments for large attachments; 0 for small attachments.
	FragmentCount uint32
}

// Blockqs is the interface boundary for the Blockqs (block query) public service
// (第 15 章 §3，DEC-0603).
//
// Blockqs provides fast querying of block transaction data, Merkle proofs, state proofs,
// recent block proof packages, and attachment indices.
// Implementation is external (github.com/cxio/blockqs); this interface defines the
// boundary contract.
//
// Blockqs is NOT a trust root: all returned data must be independently verified using
// the block header chain, CheckRoot, TxID, or attachment fingerprints.
// Clients must cross-query multiple Blockqs nodes for critical data.
//
// Service nodes supply their blockchain account address for reward distribution;
// this address declaration is not a basis for judging response authenticity.
// Service unavailability does not affect block legality.
type Blockqs interface {
	// LookupTx queries the complete transaction data for a given year and TxID.
	LookupTx(year uint64, txID types.TxID) (TxLookupResponse, error)

	// TxProof queries the Merkle proof path from a transaction to its block's tree root.
	TxProof(blockHeight uint32, txID types.TxID) (TxProofResponse, error)

	// BlockTxList queries the full TxID sequence or network summary for a block.
	// summaryMode requests the compact network summary form; full TxIDs are returned if false.
	BlockTxList(blockID types.BlockID, summaryMode bool) (BlockTxListResponse, error)

	// StateProof queries UTXO/UTCO state bit proofs for the given transaction output references.
	StateProof(txIDs []types.TxID) (StateProofResponse, error)

	// RecentBlockProofs queries the most recent block proof packages.
	// count specifies the desired number; the response must include at least
	// MinRecentBlockProofs (31) packages.
	RecentBlockProofs(count int) (RecentBlockProofsResponse, error)

	// AttachmentIndex queries the data or fragment index for a given attachment fingerprint.
	AttachmentIndex(fingerprint types.AttachmentHash) (AttachmentIndexResponse, error)
}
