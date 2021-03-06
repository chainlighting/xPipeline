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

package core

import (
	"encoding/base64"
	"fmt"
	"github.com/chainlighting/xPipeline/shared"
	"testing"
	"time"
)

type messageBatchWriter struct {
	expect  shared.Expect
	counter int
}

func (bw *messageBatchWriter) hasData(messages []Message) {
	bw.expect.Greater(len(messages), 0)
}

func (bw *messageBatchWriter) checkOrder(messages []Message) {
	for i, msg := range messages {
		bw.expect.Equal(uint64(i), msg.Sequence)
	}
}

func (bw messageBatchWriter) Write(data []byte) (int, error) {
	bw.expect.Equal("0123456789", string(data))
	return len(data), nil
}

func (bw messageBatchWriter) Flush(msg Message) {
	bw.expect.NotExecuted()
}

func (bw *messageBatchWriter) Count(msg Message) {
	bw.counter++
}

func TestMessageBatchAppendOrFlush(t *testing.T) {
	expect := shared.NewExpect(t)
	writer := messageBatchWriter{expect, 0}
	batch := NewMessageBatch(10)
	assembly := NewWriterAssembly(writer, writer.Flush, &mockFormatter{})

	flushBuffer := func() {
		batch.Flush(assembly.Write)
		batch.WaitForFlush(time.Second)
		expect.True(batch.IsEmpty())
	}

	doBlock := func() bool {
		return true
	}

	dontBlock := func() bool {
		return false
	}
	//dropMsg a stub
	dropMsg := func(msg Message) {
	}

	for i := 0; i < 10; i++ {
		batch.AppendOrFlush(NewMessage(nil, []byte(fmt.Sprintf("%d", i)), uint64(i)),
			flushBuffer,
			dontBlock,
			dropMsg)
	}
	// the buffer is full so it should be flushed and the new message queued
	batch.AppendOrFlush(NewMessage(nil, []byte(fmt.Sprintf("%d", 10)), uint64(0)),
		flushBuffer,
		doBlock,
		dropMsg)
	expect.Equal(batch.getActiveBufferCount(), int(1))

}

func TestMessageBatch(t *testing.T) {
	expect := shared.NewExpect(t)
	writer := messageBatchWriter{expect, 0}
	assembly := NewWriterAssembly(writer, writer.Flush, &mockFormatter{})

	batch := NewMessageBatch(10)
	expect.False(batch.IsClosed())
	expect.True(batch.IsEmpty())

	// length of buffer should be 10
	expect.Equal(batch.Len(), 10)

	// Append adds an item
	batch.Append(NewMessage(nil, []byte("test"), 0))
	expect.False(batch.IsEmpty())

	// Flush removes all items
	batch.Flush(writer.hasData)
	batch.WaitForFlush(time.Second)

	expect.True(batch.IsEmpty())
	expect.False(batch.ReachedSizeThreshold(10))

	// Append fails if buffer is full
	for i := 0; i < 10; i++ {
		expect.True(batch.Append(NewMessage(nil, []byte(fmt.Sprintf("%d", i)), uint64(i))))
	}
	expect.False(batch.Append(NewMessage(nil, []byte("10"), 10)))
	expect.True(batch.ReachedSizeThreshold(10))

	// Test writer assembly
	batch.Flush(assembly.Write)
	batch.WaitForFlush(time.Second)
	expect.True(batch.IsEmpty())

	for i := 0; i < 10; i++ {
		expect.True(batch.Append(NewMessage(nil, []byte(fmt.Sprintf("%d", i)), uint64(i))))
	}

	writer.counter = 0
	assembly.SetFlush(writer.Count)

	batch.Flush(assembly.Flush)
	batch.WaitForFlush(time.Second)

	expect.True(batch.IsEmpty())
	expect.Equal(10, writer.counter)

	// Flush removes all items, also if closed
	for i := 0; i < 5; i++ {
		expect.True(batch.Append(NewMessage(nil, []byte(fmt.Sprintf("%d", i)), uint64(i))))
	}

	batch.Flush(assembly.Flush)
	batch.WaitForFlush(time.Second)
	// batch is not closed and message is appended

	for i := 0; i < 10; i++ {
		expect.True(batch.AppendOrBlock(NewMessage(nil, []byte(fmt.Sprintf("%d", i)), uint64(i))))
	}
	go func() {
		expect.False(batch.AppendOrBlock(NewMessage(nil, []byte("10"), 10)))
	}()
	//let above goroutine run so that spin can Yield atleast once
	time.Sleep(1 * time.Second)

	expect.True(batch.ReachedTimeThreshold(100 * time.Millisecond))

	//closing now will close the messageBatch and the goroutine will return false
	batch.Close(writer.checkOrder, time.Second)
	expect.True(batch.IsEmpty())
	expect.False(batch.Append(NewMessage(nil, []byte("6"), 6)))
	expect.True(batch.IsEmpty())

	expect.False(batch.Append(NewMessage(nil, nil, 0)))
	expect.False(batch.AppendOrBlock(NewMessage(nil, nil, 0)))
}

func TestMessageSerialize(t *testing.T) {
	expect := shared.NewExpect(t)
	now := time.Now()

	testMessage := Message{
		StreamID:     1,
		PrevStreamID: 2,
		Timestamp:    now,
		Sequence:     4,
		Data:         []byte("This is a\nteststring"),
	}

	data, err := testMessage.Serialize()
	expect.NoError(err)
	expect.Greater(len(data), 0)

	// Test base 64 encoding of this format
	encodedSize := base64.StdEncoding.EncodedLen(len(data))
	encoded := make([]byte, encodedSize)
	base64.StdEncoding.Encode(encoded, data)

	decoded := make([]byte, base64.StdEncoding.DecodedLen(len(encoded)))
	length, _ := base64.StdEncoding.Decode(decoded, encoded)

	expect.Equal(data, decoded[:length])

	// Test deserialization
	readMessage, err := DeserializeMessage(data)
	expect.Nil(err)

	expect.Equal(readMessage.StreamID, testMessage.StreamID)
	expect.Equal(readMessage.PrevStreamID, testMessage.PrevStreamID)
	expect.Equal(readMessage.Timestamp, testMessage.Timestamp)
	expect.Equal(readMessage.Sequence, testMessage.Sequence)
	expect.Equal(readMessage.Data, testMessage.Data)
}
