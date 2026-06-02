package services

import "github.com/cxio/evidcoin/pkg/types"

// Depots is the interface boundary for the Depots (data depot) public service
// (第 15 章 §2，DEC-0603).
//
// Depots manages open-format storage and sharing of large block/transaction attachment data.
// Data queries are fulfilled via the base network's generic broadcast mechanism;
// scarcity is assessed by hop count in broadcast replies.
//
// Implementation is external (github.com/cxio/depots); this interface defines the
// boundary contract for a local blockchain node interacting with a Depots node.
//
// Depots does NOT participate in block validation, transaction execution, PoH, or
// script verification. Service unavailability does not affect block legality.
//
// Service nodes supply their blockchain account address when connecting to an application
// node to receive possible reward allocation (see proposal §14).
type Depots interface {
	// FetchAttachment requests a large attachment (>= DataBoundaryBytes) by its canonical fingerprint.
	// Returns the raw attachment bytes, or an error if unavailable.
	FetchAttachment(fingerprint types.AttachmentHash) ([]byte, error)

	// FetchBlock requests a full serialized block by its BlockID.
	// Returns the raw block bytes, or an error if unavailable.
	FetchBlock(blockID types.BlockID) ([]byte, error)

	// FetchFragment requests a single data fragment by attachment fingerprint and fragment index.
	// Returns the raw fragment bytes, or an error if unavailable.
	FetchFragment(fingerprint types.AttachmentHash, fragIndex uint32) ([]byte, error)

	// UploadAttachment uploads locally-originated attachment data to the Depots network.
	// The node acts as the data source; Depots nodes store upon receiving the query broadcast.
	// Returns an error if the upload request fails.
	UploadAttachment(fingerprint types.AttachmentHash, data []byte) error
}
