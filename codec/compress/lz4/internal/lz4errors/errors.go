// Package lz4errors defines standard error constants for LZ4 frame and block compression.
package lz4errors

// Error represents an LZ4 frame or block error condition.
type Error string

func (e Error) Error() string { return string(e) }

const (
	// ErrInvalidSourceShortBuffer indicates that the source or destination buffer is too short.
	ErrInvalidSourceShortBuffer Error = "lz4: invalid source or destination buffer too short"
	// ErrInvalidFrame indicates that the stream header contains a bad magic number.
	ErrInvalidFrame Error = "lz4: bad magic number"
	// ErrInternalUnhandledState indicates an unhandled internal decompression state.
	ErrInternalUnhandledState Error = "lz4: unhandled state"
	// ErrInvalidHeaderChecksum indicates that the frame header checksum is invalid.
	ErrInvalidHeaderChecksum Error = "lz4: invalid header checksum"
	// ErrInvalidBlockChecksum indicates that a block checksum verification failed.
	ErrInvalidBlockChecksum Error = "lz4: invalid block checksum"
	// ErrInvalidFrameChecksum indicates that the frame trailing checksum verification failed.
	ErrInvalidFrameChecksum Error = "lz4: invalid frame checksum"
	// ErrOptionInvalidCompressionLevel indicates that an invalid compression level was specified.
	ErrOptionInvalidCompressionLevel Error = "lz4: invalid compression level"
	// ErrOptionClosedOrError indicates that an option cannot be applied to a closed or failed stream.
	ErrOptionClosedOrError Error = "lz4: cannot apply options on closed or in error object"
	// ErrOptionInvalidBlockSize indicates that an invalid block size option was specified.
	ErrOptionInvalidBlockSize Error = "lz4: invalid block size"
	// ErrOptionNotApplicable indicates that the option cannot be applied to this object.
	ErrOptionNotApplicable Error = "lz4: option not applicable"
	// ErrWriterNotClosed indicates that the LZ4 writer was not properly closed.
	ErrWriterNotClosed Error = "lz4: writer not closed"
	// ErrEndOfStream indicates that the end of the LZ4 stream was reached.
	ErrEndOfStream Error = "lz4: end of stream reached"
)
