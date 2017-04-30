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
	"fmt"
	"github.com/chainlighting/xPipeline/shared"
	"sync"
	"testing"
)

func getMockStreamRegistry() streamRegistry {
	return streamRegistry{
		streams:     map[MessageStreamID]Stream{},
		name:        map[MessageStreamID]string{},
		fuses:       map[string]*shared.Fuse{},
		streamGuard: new(sync.Mutex),
		nameGuard:   new(sync.Mutex),
		fuseGuard:   new(sync.Mutex),
		wildcard:    []Producer{},
	}
}

func mockDistributer(msg Message) {

}

func mockPrevDistributer(msg Message) {

}

func resetStreamRegistryCounts() {
	messageCount = 0
	droppedCount = 0
	discardedCount = 0
	filteredCount = 0
	noRouteCount = 0
}

func TestStreamRegistryAtomicOperations(t *testing.T) {
	expect := shared.NewExpect(t)
	// Tests run in one go, so the counts might already be affected. Reset here.
	resetStreamRegistryCounts()

	expect.Equal(messageCount, uint32(0))
	CountProcessedMessage()
	expect.Equal(messageCount, uint32(1))

	expect.Equal(droppedCount, uint32(0))
	CountDroppedMessage()
	expect.Equal(droppedCount, uint32(1))

	expect.Equal(discardedCount, uint32(0))
	CountDiscardedMessage()
	expect.Equal(discardedCount, uint32(1))

	expect.Equal(filteredCount, uint32(0))
	CountFilteredMessage()
	expect.Equal(filteredCount, uint32(1))

	expect.Equal(noRouteCount, uint32(0))
	CountNoRouteForMessage()
	expect.Equal(noRouteCount, uint32(1))

	messages, dropped, discarded, filtered, noroute := GetAndResetMessageCount()

	expect.Equal(messages, uint32(1))
	expect.Equal(dropped, uint32(1))
	expect.Equal(discarded, uint32(1))
	expect.Equal(filtered, uint32(1))
	expect.Equal(noroute, uint32(1))

	expect.Equal(messageCount, uint32(0))
	expect.Equal(droppedCount, uint32(0))
	expect.Equal(discardedCount, uint32(0))
	expect.Equal(filteredCount, uint32(0))
	expect.Equal(noRouteCount, uint32(0))
}

func TestStreamRegistryGetStreamName(t *testing.T) {
	expect := shared.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	expect.Equal(mockSRegistry.GetStreamName(DroppedStreamID), DroppedStream)
	expect.Equal(mockSRegistry.GetStreamName(LogInternalStreamID), LogInternalStream)
	expect.Equal(mockSRegistry.GetStreamName(WildcardStreamID), WildcardStream)

	mockStreamID := StreamRegistry.GetStreamID("test")
	expect.Equal(mockSRegistry.GetStreamName(mockStreamID), "")

	mockSRegistry.name[mockStreamID] = "test"
	expect.Equal(mockSRegistry.GetStreamName(mockStreamID), "test")
}

func TestStreamRegistryGetStreamByName(t *testing.T) {
	expect := shared.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	streamName := mockSRegistry.GetStreamByName("testStream")
	expect.Equal(streamName, nil)

	mockStreamID := StreamRegistry.GetStreamID("testStream")
	// TODO: Get a real stream and test with that
	mockSRegistry.streams[mockStreamID] = &StreamBase{}
	expect.Equal(mockSRegistry.GetStreamByName("testStream"), &StreamBase{})
}

func TestStreamRegistryIsStreamRegistered(t *testing.T) {
	expect := shared.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	mockStreamID := StreamRegistry.GetStreamID("testStream")

	expect.False(mockSRegistry.IsStreamRegistered(mockStreamID))
	// TODO: Get a real stream and test with that
	mockSRegistry.streams[mockStreamID] = &StreamBase{}
	expect.True(mockSRegistry.IsStreamRegistered(mockStreamID))
}

func TestStreamRegistryForEachStream(t *testing.T) {
	expect := shared.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	callback := func(streamID MessageStreamID, stream Stream) {
		expect.Equal(streamID, StreamRegistry.GetStreamID("testRegistry"))
	}

	mockSRegistry.streams[StreamRegistry.GetStreamID("testRegistry")] = &StreamBase{}
	mockSRegistry.ForEachStream(callback)
}

func TestStreamRegistryWildcardProducer(t *testing.T) {
	expect := shared.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()
	// WildcardProducersExist()
	expect.False(mockSRegistry.WildcardProducersExist())

	producer1 := new(mockProducer)
	producer2 := new(mockProducer)

	mockSRegistry.RegisterWildcardProducer(producer1, producer2)

	expect.True(mockSRegistry.WildcardProducersExist())
}

func TestStreamRegistryAddWildcardProducersToStream(t *testing.T) {
	expect := shared.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	// create stream to which wildcardProducer is to be added
	mockStream := getMockStream()

	// create wildcardProducer.
	mProducer := new(mockProducer)
	// adding dropStreamID to verify the producer later.
	mProducer.dropStreamID = StreamRegistry.GetStreamID("wildcardProducerDrop")
	mockSRegistry.RegisterWildcardProducer(mProducer)

	mockSRegistry.AddWildcardProducersToStream(&mockStream)

	streamsProducer := mockStream.GetProducers()
	expect.Equal(len(streamsProducer), 1)

	expect.Equal(streamsProducer[0].GetDropStreamID(), StreamRegistry.GetStreamID("wildcardProducerDrop"))
}

func TestStreamRegistryRegister(t *testing.T) {
	expect := shared.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	streamName := "testStream"
	mockStream := getMockStream()
	mockSRegistry.Register(&mockStream, StreamRegistry.GetStreamID(streamName))

	expect.NotNil(mockSRegistry.GetStream(StreamRegistry.GetStreamID(streamName)))
}

func TestStreamRegistryGetStreamOrFallback(t *testing.T) {
	expect := shared.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	expect.Equal(len(mockSRegistry.streams), 0)
	expect.Equal(len(mockSRegistry.wildcard), 0)

	streamName := "testStream"
	streamID := StreamRegistry.GetStreamID(streamName)
	mockSRegistry.GetStreamOrFallback(streamID)

	expect.Equal(len(mockSRegistry.streams), 1)

	// try registering again. No new register should happen.
	mockSRegistry.GetStreamOrFallback(streamID)
	expect.Equal(len(mockSRegistry.streams), 1)
}

func TestStreamRegistryLinkDependencies(t *testing.T) {
	/**
	The dependency tree in this test is given below. A -> B means A depends on B
		producer1 -> producer3
		producer3 -> producer2
		producer2 -> none
		producer4 -> none
	*/
	expect := shared.NewExpect(t)
	mockSRegistry := getMockStreamRegistry()

	streamName := "testStream"
	streamID := StreamRegistry.GetStreamID(streamName)
	//register a stream
	stream := mockSRegistry.GetStreamOrFallback(streamID)

	producer1 := mockProducer{}
	producer2 := mockProducer{}
	producer3 := mockProducer{}
	producer4 := mockProducer{}

	expect.Equal(len(producer1.dependencies), 0)
	expect.Equal(len(producer2.dependencies), 0)
	expect.Equal(len(producer3.dependencies), 0)
	expect.Equal(len(producer4.dependencies), 0)

	producer1.AddDependency(&producer3)
	producer3.AddDependency(&producer2)

	stream.AddProducer(&producer1, &producer2, &producer3, &producer4)

	mockSRegistry.LinkDependencies(&producer3, streamID)

	expect.Equal(len(producer1.dependencies), 1)
	expect.Equal(len(producer2.dependencies), 0)
	expect.Equal(len(producer3.dependencies), 1)
	expect.Equal(len(producer4.dependencies), 1)

}

func TestStreamRegistryConcurrency(t *testing.T) {
	mockSRegistry := getMockStreamRegistry()
	expect := shared.NewExpect(t)
	routines := sync.WaitGroup{}

	for i := 0; i < 10; i++ {
		routines.Add(1)
		base := i
		go func() {
			for j := 0; j < 100; j++ {
				streamID := mockSRegistry.GetStreamID(fmt.Sprintf("test%d", base*100+j))
				mockSRegistry.GetStreamOrFallback(streamID)
			}
			routines.Done()
		}()
	}

	routines.Wait()
	expect.Equal(len(mockSRegistry.streams), len(mockSRegistry.name))
}
