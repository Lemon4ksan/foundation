// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package qpack

// StreamReceiver decodes QPACK data received on a unidirectional control stream.
// Direct 1:1 structural translation of Chromium's quiche::StreamReceiver.
type StreamReceiver interface {
	// Decode decodes data received on the control stream.
	Decode(data []byte)
}

// EncoderStreamReceiverDelegate handles instructions decoded from the encoder stream (RFC 9204 §4.3).
// Direct 1:1 structural translation of Chromium's quiche::EncoderStreamReceiver::Delegate.
type EncoderStreamReceiverDelegate interface {
	// InsertWithNameReference is called when an Insert With Name Reference instruction is decoded (RFC 9204 §4.3.1).
	InsertWithNameReference(isStatic bool, nameIndex uint64, value string)

	// InsertWithoutNameReference is called when an Insert Without Name Reference instruction is decoded (RFC 9204 §4.3.2).
	InsertWithoutNameReference(name, value string)

	// Duplicate is called when a Duplicate instruction is decoded (RFC 9204 §4.3.3).
	Duplicate(index uint64)

	// SetDynamicTableCapacity is called when a Set Dynamic Table Capacity instruction is decoded (RFC 9204 §4.3.4).
	SetDynamicTableCapacity(capacity uint64)

	// Error is called when an instruction decoding error or stream error is detected.
	Error(qpackError uint64, errorMessage string)
}

// DecoderStreamReceiverDelegate handles instructions decoded from the decoder stream (RFC 9204 §4.4).
// Direct 1:1 structural translation of Chromium's quiche::DecoderStreamReceiver::Delegate.
type DecoderStreamReceiverDelegate interface {
	// SectionAck is called when a Section Acknowledgment instruction is decoded (RFC 9204 §4.4.1).
	// Corresponds to Chromium's OnHeaderAcknowledgement.
	SectionAck(streamID uint64)

	// StreamCancellation is called when a Stream Cancellation instruction is decoded (RFC 9204 §4.4.2).
	// Corresponds to Chromium's OnStreamCancellation.
	StreamCancellation(streamID uint64)

	// InsertCountIncrement is called when an Insert Count Increment instruction is decoded (RFC 9204 §4.4.3).
	// Corresponds to Chromium's OnInsertCountIncrement.
	InsertCountIncrement(increment uint64)

	// Error is called when an instruction decoding error or stream error is detected.
	// Corresponds to Chromium's OnErrorDetected.
	Error(qpackError uint64, errorMessage string)
}

// EncoderStreamReceiver decodes data received on the encoder stream.
// Direct 1:1 structural translation of Chromium's quiche::EncoderStreamReceiver.
type EncoderStreamReceiver struct {
	instructionDecoder *InstructionDecoder
	delegate           EncoderStreamReceiverDelegate
	errorDetected      bool
}

// NewEncoderStreamReceiver creates a new encoder stream receiver.
func NewEncoderStreamReceiver(delegate EncoderStreamReceiverDelegate) *EncoderStreamReceiver {
	if delegate == nil {
		panic("qpack: delegate must not be nil")
	}
	r := &EncoderStreamReceiver{
		delegate: delegate,
	}
	r.instructionDecoder = NewInstructionDecoder(EncoderStreamLanguage(), r)
	return r
}

// Decode feeds data into the instruction decoder.
// Once an error occurs, delegate.Error() is called, and all further data is ignored.
func (r *EncoderStreamReceiver) Decode(data []byte) {
	if len(data) == 0 || r.errorDetected {
		return
	}
	r.instructionDecoder.Decode(data)
}

// EndDecoding signals that no more data is expected on the stream.
func (r *EncoderStreamReceiver) EndDecoding() {
	if r.errorDetected {
		return
	}
	r.instructionDecoder.EndDecoding()
}

// OnInstructionDecoded implements InstructionDecoderDelegate.
func (r *EncoderStreamReceiver) OnInstructionDecoded(instruction *Instruction) bool {
	if instruction == InsertWithNameReferenceInstruction() {
		r.delegate.InsertWithNameReference(
			r.instructionDecoder.SBit(),
			r.instructionDecoder.Varint(),
			r.instructionDecoder.Value(),
		)
		return true
	}

	if instruction == InsertWithoutNameReferenceInstruction() {
		r.delegate.InsertWithoutNameReference(
			r.instructionDecoder.Name(),
			r.instructionDecoder.Value(),
		)
		return true
	}

	if instruction == DuplicateInstruction() {
		r.delegate.Duplicate(r.instructionDecoder.Varint())
		return true
	}

	if instruction == SetDynamicTableCapacityInstruction() {
		r.delegate.SetDynamicTableCapacity(r.instructionDecoder.Varint())
		return true
	}

	return false
}

// OnInstructionDecodingError implements InstructionDecoderDelegate.
func (r *EncoderStreamReceiver) OnInstructionDecodingError(
	errorCode InstructionDecoderErrorCode,
	errorMessage string,
) {
	if r.errorDetected {
		return
	}
	r.errorDetected = true
	r.delegate.Error(QPACK_ENCODER_STREAM_ERROR, errorMessage)
}

// DecoderStreamReceiver decodes data received on the decoder stream.
// Direct 1:1 structural translation of Chromium's quiche::DecoderStreamReceiver.
type DecoderStreamReceiver struct {
	instructionDecoder *InstructionDecoder
	delegate           DecoderStreamReceiverDelegate
	errorDetected      bool
}

// NewDecoderStreamReceiver creates a new decoder stream receiver.
func NewDecoderStreamReceiver(delegate DecoderStreamReceiverDelegate) *DecoderStreamReceiver {
	if delegate == nil {
		panic("qpack: delegate must not be nil")
	}
	r := &DecoderStreamReceiver{
		delegate: delegate,
	}
	r.instructionDecoder = NewInstructionDecoder(DecoderStreamLanguage(), r)
	return r
}

// Decode feeds data into the instruction decoder.
// Once an error occurs, delegate.Error() is called, and all further data is ignored.
func (r *DecoderStreamReceiver) Decode(data []byte) {
	if len(data) == 0 || r.errorDetected {
		return
	}
	r.instructionDecoder.Decode(data)
}

// EndDecoding signals that no more data is expected on the stream.
func (r *DecoderStreamReceiver) EndDecoding() {
	if r.errorDetected {
		return
	}
	r.instructionDecoder.EndDecoding()
}

// OnInstructionDecoded implements InstructionDecoderDelegate.
func (r *DecoderStreamReceiver) OnInstructionDecoded(instruction *Instruction) bool {
	if instruction == InsertCountIncrementInstruction() {
		r.delegate.InsertCountIncrement(r.instructionDecoder.Varint())
		return true
	}

	if instruction == HeaderAcknowledgementInstruction() {
		r.delegate.SectionAck(r.instructionDecoder.Varint())
		return true
	}

	if instruction == StreamCancellationInstruction() {
		r.delegate.StreamCancellation(r.instructionDecoder.Varint())
		return true
	}

	return false
}

// OnInstructionDecodingError implements InstructionDecoderDelegate.
func (r *DecoderStreamReceiver) OnInstructionDecodingError(
	errorCode InstructionDecoderErrorCode,
	errorMessage string,
) {
	if r.errorDetected {
		return
	}
	r.errorDetected = true
	r.delegate.Error(QPACK_DECODER_STREAM_ERROR, errorMessage)
}

// QuicErrorCodeFromEncoderInstructionError returns Chromium's QuicErrorCode for an encoder stream error.
func QuicErrorCodeFromEncoderInstructionError(errCode InstructionDecoderErrorCode) uint64 {
	switch errCode {
	case InstructionDecoderIntegerTooLarge:
		return QUIC_QPACK_ENCODER_STREAM_INTEGER_TOO_LARGE
	case InstructionDecoderStringLiteralTooLong:
		return QUIC_QPACK_ENCODER_STREAM_STRING_LITERAL_TOO_LONG
	case InstructionDecoderHuffmanEncodingError:
		return QUIC_QPACK_ENCODER_STREAM_HUFFMAN_ENCODING_ERROR
	default:
		return 0
	}
}

// QuicErrorCodeFromDecoderInstructionError returns Chromium's QuicErrorCode for a decoder stream error.
func QuicErrorCodeFromDecoderInstructionError(errCode InstructionDecoderErrorCode) uint64 {
	if errCode == InstructionDecoderIntegerTooLarge {
		return QUIC_QPACK_DECODER_STREAM_INTEGER_TOO_LARGE
	}
	return 0
}

// Interface compliance assertions
var (
	_ StreamReceiver             = (*EncoderStreamReceiver)(nil)
	_ StreamReceiver             = (*DecoderStreamReceiver)(nil)
	_ InstructionDecoderDelegate = (*EncoderStreamReceiver)(nil)
	_ InstructionDecoderDelegate = (*DecoderStreamReceiver)(nil)
)
