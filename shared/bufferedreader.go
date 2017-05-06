// Copyright 2015-2016 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package shared

import (
	"bytes"
	"encoding/binary"
	"io"
)

// BufferedReaderFlags is an enum to configure a buffered reader
type BufferedReaderFlags byte

const (
	// BufferedReaderFlagDelimiter enables reading for a delimiter. This flag is
	// ignored if an MLE flag is set.
	BufferedReaderFlagDelimiter = BufferedReaderFlags(0)

	// BufferedReaderFlagMLE enables reading if length encoded messages.
	// Runlength is read as ASCII (to uint64) until the first byte (ASCII char)
	// of the delimiter string.
	// Only one MLE flag is supported at a time.
	BufferedReaderFlagMLE = BufferedReaderFlags(1)

	// BufferedReaderFlagMLE8 enables reading if length encoded messages.
	// Runlength is read as binary (to uint8).
	// Only one MLE flag is supported at a time.
	BufferedReaderFlagMLE8 = BufferedReaderFlags(2)

	// BufferedReaderFlagMLE16 enables reading if length encoded messages.
	// Runlength is read as binary (to uint16).
	// Only one MLE flag is supported at a time.
	BufferedReaderFlagMLE16 = BufferedReaderFlags(3)

	// BufferedReaderFlagMLE32 enables reading if length encoded messages.
	// Runlength is read as binary (to uint32).
	// Only one MLE flag is supported at a time.
	BufferedReaderFlagMLE32 = BufferedReaderFlags(4)

	// BufferedReaderFlagMLE64 enables reading if length encoded messages.
	// Runlength is read as binary (to uint64).
	// Only one MLE flag is supported at a time.
	BufferedReaderFlagMLE64 = BufferedReaderFlags(5)

	// BufferedReaderFlagMLEFixed enables reading messages with a fixed length.
	// Only one MLE flag is supported at a time.
	BufferedReaderFlagMLEFixed = BufferedReaderFlags(6)

	// BufferedReaderFlagMaskMLE is a bitmask to mask out everything but MLE flags
	BufferedReaderFlagMaskMLE = BufferedReaderFlags(7)

	// BufferedReaderFlagBigEndian sets binary reading to big endian encoding.
	BufferedReaderFlagBigEndian = BufferedReaderFlags(8)

	// BufferedReaderFlagEverything will keep MLE and/or delimiters when
	// building a message.
	BufferedReaderFlagEverything = BufferedReaderFlags(16)
)

type bufferError string

func (b bufferError) Error() string {
	return string(b)
}

// BufferReadCallback defines the function signature for callbacks passed to
// ReadAll.
type BufferReadCallback func(msg []byte, sequence uint64)

// MsgCheckCompleteCallback defines the function for split a completed msg from bytestream.
// parseComplex.
type MsgCheckCompleteCallback func(data []byte) (int,int,int)

// BufferDataInvalid is returned when a parsing encounters an error
var BufferDataInvalid = bufferError("Invalid data")

// BufferedReader is a helper struct to read from any io.Reader into a byte
// slice. The data can arrive "in pieces" and will be assembled.
// A data "piece" is considered complete if a delimiter or a certain runlength
// has been reached. The latter has to be enabled by flag and will disable the
// default behavior, which is looking for a delimiter string.
// In addition to that every data "piece" will receive an incrementing sequence
// number.
type BufferedReader struct {
	data       []byte
	delimiter  []byte
	parse      func() ([]byte)
	checkComplete MsgCheckCompleteCallback
	sequence   uint64
	paramMLE   int
	growSize   int
	begin      int
	end        int
	encoding   binary.ByteOrder
	flags      BufferedReaderFlags
	incomplete bool
}

// NewBufferedReader creates a new buffered reader that reads messages from a
// continuous stream of bytes.
// Messages can be separated from the stream by using common methods such as
// fixed size, encoded message length or delimiter string.
// The internal buffer is grown statically (by its original size) if necessary.
// bufferSize defines the initial size / grow size of the buffer
// flags configures the parsing method
// offsetOrLength sets either the runlength offset or fixed message size
// delimiter defines the delimiter used for textual message parsing
func NewBufferedReader(bufferSize int, flags BufferedReaderFlags, offsetOrLength int, delimiter string,checkComplete MsgCheckCompleteCallback) *BufferedReader {
	buffer := BufferedReader{
		data:       make([]byte, bufferSize),
		delimiter:  []byte(delimiter),
		checkComplete: checkComplete,
		paramMLE:   offsetOrLength,
		encoding:   binary.LittleEndian,
		sequence:   0,
		begin:      0,
		end:        0,
		flags:      flags,
		growSize:   bufferSize,
		incomplete: true,
	}

    // simple msg parse setting
	if flags&BufferedReaderFlagBigEndian != 0 {
		buffer.encoding = binary.BigEndian
	}

	if nil == buffer.checkComplete {
		if 0 == flags&BufferedReaderFlagMaskMLE {
			buffer.checkComplete = buffer.checkCompleteDelimiter
		} else {
			switch flags & BufferedReaderFlagMaskMLE {
			default:
				buffer.checkComplete = buffer.checkCompleteMLEText
			case BufferedReaderFlagMLE8:
				buffer.checkComplete = buffer.checkCompleteMLE8
			case BufferedReaderFlagMLE16:
				buffer.checkComplete = buffer.checkCompleteMLE16
			case BufferedReaderFlagMLE32:
				buffer.checkComplete = buffer.checkCompleteMLE32
			case BufferedReaderFlagMLE64:
				buffer.checkComplete = buffer.checkCompleteMLE64
			case BufferedReaderFlagMLEFixed:
				buffer.checkComplete = buffer.checkCompleteMLEFixed
			}
		}
	}

	buffer.parse = buffer.parseComplex

	return &buffer
}

// Reset clears the buffer by resetting its internal state
func (buffer *BufferedReader) Reset(sequence uint64) {
	buffer.sequence = sequence
	buffer.begin = 0
	buffer.end = 0
	buffer.incomplete = true
}

// general message extraction part of all parser methods
// retrun msgData []byte
func (buffer *BufferedReader) extractMessage(msgLen int, msgStartOffset int,padLen int) ([]byte) {
    msgNextBegin := buffer.begin + msgStartOffset + msgLen + padLen
    if msgNextBegin > buffer.end {
		return nil // ### return, incomplete ###
	}
	dataRtn := buffer.data[buffer.begin + msgStartOffset:buffer.begin + msgStartOffset + msgLen]
	buffer.begin = msgNextBegin
	return dataRtn
}

// messages parsed by external MsgCheckCompleteCallback
// return: msgData []byte
func (buffer *BufferedReader) parseComplex() ([]byte) {
	if buffer.checkComplete != nil {
		msgLen, msgStartOffset, padLen := buffer.checkComplete(buffer.data[buffer.begin:buffer.end])
		if msgLen > 0 {
			// msg completed
			return buffer.extractMessage(msgLen,msgStartOffset,padLen)
		}
	}

	// no completed msg
	return nil
}

// messages have a fixed size
func (buffer *BufferedReader) checkCompleteMLEFixed(data []byte) (int,int,int) {
	if len(data) >= buffer.paramMLE {
		return buffer.paramMLE,0,0
	}
	// not complete
	return 0,0,0
}

// messages are separated by a delimiter string
func (buffer *BufferedReader) checkCompleteDelimiter(data []byte) (int,int,int) {
	delimiterIdx := bytes.Index(data, buffer.delimiter)
	if delimiterIdx == -1 {
		return 0, 0, 0 // ### return, incomplete ###
	}
	msgLen := delimiterIdx
	padLen := len(buffer.delimiter)
	if buffer.flags&BufferedReaderFlagEverything != 0 {
		msgLen += padLen
		padLen = 0
	}
	return msgLen,0,padLen
}

// messages are separeated length encoded by ASCII number and (an optional)
// delimiter.
func (buffer *BufferedReader) checkCompleteMLEText(data []byte) (int,int,int) {
	if len(data) <= buffer.paramMLE {
		return 0, 0, 0
	}
	msgLen, msgStartOffset := Btoi(data[buffer.paramMLE:])
	if msgStartOffset == 0 {
		return 0, 0, 0 // ### return, malformed ###
	}

	msgStartOffset += buffer.paramMLE
	// Read delimiter if necessary (check if valid runlength)
	padLen := len(buffer.delimiter)
	if padLen > 0 {
		msgStartOffset += padLen
		if !bytes.Equal(data[msgStartOffset-padLen:msgStartOffset], buffer.delimiter) {
			return 0, 0, 0 // ### return, malformed ###
		}
	}

	if (int(msgLen) + msgStartOffset) > len(data) {
		return 0, 0, 0 // incomplete
	}

	if buffer.flags&BufferedReaderFlagEverything != 0 {
		return int(msgLen) + msgStartOffset,0,0
	} else {
		return int(msgLen), msgStartOffset,0
	}
}

// messages are separated binary length encoded
func (buffer *BufferedReader) checkCompleteMLE8(data []byte) (int,int,int) {
	var msgLen uint8
	reader := bytes.NewReader(data[buffer.paramMLE:])
	err := binary.Read(reader, buffer.encoding, &msgLen)
	if err != nil {
		return 0, -1, 0 // ### return, malformed ###
	}

    msgStartOffset := buffer.paramMLE + 1
	if (int(msgLen) + msgStartOffset) > len(data) {
		return 0, 0, 0 // incomplete
	}    
	if buffer.flags&BufferedReaderFlagEverything != 0 {
		return int(msgLen) + msgStartOffset,0,0
	} else {
		return int(msgLen), msgStartOffset,0
	}
}

// messages are separated binary length encoded
func (buffer *BufferedReader) checkCompleteMLE16(data []byte) (int,int,int) {
	var msgLen uint16
	reader := bytes.NewReader(data[buffer.paramMLE:])
	err := binary.Read(reader, buffer.encoding, &msgLen)
	if err != nil {
		return 0, -1, 0 // ### return, malformed ###
	}

	msgStartOffset := buffer.paramMLE + 2
	if (int(msgLen) + msgStartOffset) > len(data) {
		return 0, 0, 0 // incomplete
	}    
	if buffer.flags&BufferedReaderFlagEverything != 0 {
		return int(msgLen) + msgStartOffset,0,0
	} else {
		return int(msgLen), msgStartOffset,0
	}
}

// messages are separated binary length encoded
func (buffer *BufferedReader) checkCompleteMLE32(data []byte) (int,int,int) {
	var msgLen uint32
	reader := bytes.NewReader(data[buffer.paramMLE:])
	err := binary.Read(reader, buffer.encoding, &msgLen)
	if err != nil {
		return 0, -1, 0 // ### return, malformed ###
	}

	msgStartOffset := buffer.paramMLE + 4
	if (int(msgLen) + msgStartOffset) > len(data) {
		return 0, 0, 0 // incomplete
	}    
	if buffer.flags&BufferedReaderFlagEverything != 0 {
		return int(msgLen) + msgStartOffset,0,0
	} else {
		return int(msgLen), msgStartOffset,0
	}
}

// messages are separated binary length encoded
func (buffer *BufferedReader) checkCompleteMLE64(data []byte) (int,int,int) {
	var msgLen uint64
	reader := bytes.NewReader(data[buffer.paramMLE:])
	err := binary.Read(reader, buffer.encoding, &msgLen)
	if err != nil {
		return 0, -1, 0 // ### return, malformed ###
	}

	msgStartOffset := buffer.paramMLE + 8
	if (int(msgLen) + msgStartOffset) > len(data) {
		return 0, 0, 0 // incomplete
	}    
	if buffer.flags&BufferedReaderFlagEverything != 0 {
		return int(msgLen) + msgStartOffset,0,0
	} else {
		return int(msgLen), msgStartOffset,0
	}
}

// ReadAll calls ReadOne as long as there are messages in the stream.
// Messages will be send to the given write callback.
// If callback is nil, data will be read and discarded.
func (buffer *BufferedReader) ReadAll(reader io.Reader, callback BufferReadCallback) error {
	for {
		data, seq, more, err := buffer.ReadOne(reader)
		if data != nil && callback != nil {
			callback(data, seq)
		}

		if err != nil {
			return err // ### return, error ###
		}

		if !more {
			return nil // ### return, done ###
		}
	}
}

// ReadOne reads the next message from the given stream (if possible) and
// generates a sequence number for this message.
// The more return parameter is set to true if there are still messages or parts
// of messages in the stream. Data and seq is only set if a complete message
// could be parsed.
// Errors are returned if reading from the stream failed or the parser
// encountered an error.
func (buffer *BufferedReader) ReadOne(reader io.Reader) (data []byte, seq uint64, more bool, err error) {
	if buffer.incomplete {
		bytesRead, err := reader.Read(buffer.data[buffer.end:])

		if err != nil && err != io.EOF {
			return nil, 0, (buffer.end - buffer.begin) > 0, err // ### return, error reading ###
		}

		if bytesRead == 0 {
			return nil, 0, (buffer.end - buffer.begin) > 0, err // ### return, no data ###
		}

		buffer.end += bytesRead
		buffer.incomplete = false
	}

	msgData := buffer.parse()

	if msgData == nil {
		// check empty
		if buffer.begin == buffer.end {
			buffer.Reset(buffer.sequence)
			return nil,0,true,err
		}
		// Check if buffer needs to be resized
		if len(buffer.data) == buffer.end {
			if 0 != buffer.begin {
				copy(buffer.data, buffer.data[buffer.begin:buffer.end])				
				buffer.end = buffer.end - buffer.begin
				buffer.begin = 0
			} else {
			    temp := buffer.data
			    buffer.data = make([]byte, len(buffer.data)+buffer.growSize)
			    copy(buffer.data, temp)
			}
		}
		buffer.incomplete = true
		return nil, 0, true, err // ### return, incomplete ###
	}

	msgDataCopy := make([]byte, len(msgData))
	copy(msgDataCopy, msgData)

	if (0 != buffer.begin)&&(buffer.begin == buffer.end) {
		buffer.Reset(buffer.sequence)
	}

	seqNum := buffer.sequence
	buffer.sequence++
	return msgDataCopy, seqNum, (buffer.end - buffer.begin) > 0, err
}
