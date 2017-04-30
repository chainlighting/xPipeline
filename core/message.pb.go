// Code generated by protoc-gen-go.
// source: message.proto
// DO NOT EDIT!

/*
Package core is a generated protocol buffer package.

It is generated from these files:
	message.proto

It has these top-level messages:
	SerializedMessage
*/
package core

import proto "github.com/golang/protobuf/proto"
import "github.com/chainlighting/xPipeline/shared"
import math "math"

// Reference imports to suppress errors if they are not otherwise used.
var _ = proto.Marshal
var _ = math.Inf

type SerializedMessage struct {
	StreamID         *uint64 `protobuf:"varint,1,req" json:"StreamID,omitempty"`
	PrevStreamID     *uint64 `protobuf:"varint,2,req" json:"PrevStreamID,omitempty"`
	Timestamp        *int64  `protobuf:"varint,3,req" json:"Timestamp,omitempty"`
	Sequence         *uint64 `protobuf:"varint,4,req" json:"Sequence,omitempty"`
	Data             []byte  `protobuf:"bytes,5,req" json:"Data,omitempty"`
	XXX_unrecognized []byte  `json:"-"`
}

func (m *SerializedMessage) Reset()         { *m = SerializedMessage{} }
func (m *SerializedMessage) String() string { return proto.CompactTextString(m) }
func (*SerializedMessage) ProtoMessage()    {}

func (m *SerializedMessage) GetStreamID() uint64 {
	if m != nil && m.StreamID != nil {
		return *m.StreamID
	}
	return 0
}

func (m *SerializedMessage) GetPrevStreamID() uint64 {
	if m != nil && m.PrevStreamID != nil {
		return *m.PrevStreamID
	}
	return 0
}

func (m *SerializedMessage) GetTimestamp() int64 {
	if m != nil && m.Timestamp != nil {
		return *m.Timestamp
	}
	return 0
}

func (m *SerializedMessage) GetSequence() uint64 {
	if m != nil && m.Sequence != nil {
		return *m.Sequence
	}
	return 0
}

func (m *SerializedMessage) GetData() []byte {
	if m != nil {
		return m.Data
	}
	return nil
}

func init() {
	shared.TypeRegistry.Register(SerializedMessage{})
}
