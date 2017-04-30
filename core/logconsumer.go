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
	"github.com/chainlighting/xPipeline/shared"
	"strings"
	"sync"
	"time"
)

// LogConsumer is an internal consumer plugin used indirectly by the gollum log
// package.
type LogConsumer struct {
	Consumer
	control        chan PluginControl
	logStream      Stream
	sequence       uint64
	metric         string
	lastCount      int64
	lastCountWarn  int64
	lastCountError int64
	updateTimer    *time.Timer
}

// Configure initializes this consumer with values from a plugin config.
func (cons *LogConsumer) Configure(conf PluginConfig) error {
	cons.control = make(chan PluginControl, 1)
	cons.logStream = StreamRegistry.GetStream(LogInternalStreamID)
	cons.metric = conf.GetString("MetricKey", "")

	if cons.metric != "" {
		shared.Metric.New(cons.metric)
		shared.Metric.New(cons.metric + "Error")
		shared.Metric.New(cons.metric + "Warning")
		shared.Metric.New(cons.metric + "Sec")
		shared.Metric.New(cons.metric + "ErrorSec")
		shared.Metric.New(cons.metric + "WarningSec")
		cons.updateTimer = time.AfterFunc(time.Second*5, cons.updateMetric)
	}
	return nil
}

// GetState always returns PluginStateActive
func (cons *LogConsumer) GetState() PluginState {
	return PluginStateActive
}

// Streams always returns an array with one member - the internal log stream
func (cons *LogConsumer) Streams() []MessageStreamID {
	return []MessageStreamID{LogInternalStreamID}
}

func (cons *LogConsumer) updateMetric() {
	currentCount, _ := shared.Metric.Get(cons.metric)
	currentCountWarn, _ := shared.Metric.Get(cons.metric + "Warning")
	currentCountError, _ := shared.Metric.Get(cons.metric + "Error")

	shared.Metric.Set(cons.metric+"Sec", (currentCount-cons.lastCount)/5)
	shared.Metric.Set(cons.metric+"WarningSec", (currentCountWarn-cons.lastCountWarn)/5)
	shared.Metric.Set(cons.metric+"ErrorSec", (currentCountError-cons.lastCountError)/5)

	cons.lastCount = currentCount
	cons.lastCountWarn = currentCountWarn
	cons.lastCountError = currentCountError
	cons.updateTimer = time.AfterFunc(time.Second*5, cons.updateMetric)
}

// Write fulfills the io.Writer interface
func (cons LogConsumer) Write(data []byte) (int, error) {
	dataCopy := make([]byte, len(data))
	copy(dataCopy, data)

	msg := NewMessage(cons, dataCopy, cons.sequence)
	msg.StreamID = LogInternalStreamID
	cons.logStream.Enqueue(msg)

	if cons.metric != "" {
		// HACK: Use different writers with the possibility to enable/disable metrics
		switch {
		case strings.HasPrefix(string(data), "ERROR"):
			shared.Metric.Inc(cons.metric + "Error")
		case strings.HasPrefix(string(data), "Warning"):
			shared.Metric.Inc(cons.metric + "Warning")
		default:
			shared.Metric.Inc(cons.metric)
		}
	}

	return len(data), nil
}

// Control returns a handle to the control channel
func (cons *LogConsumer) Control() chan<- PluginControl {
	return cons.control
}

// Consume starts listening for control statements
func (cons *LogConsumer) Consume(threads *sync.WaitGroup) {
	// Wait for control statements
	for {
		command := <-cons.control
		if command == PluginControlStopConsumer {
			cons.updateTimer.Stop()
			return // ### return ###
		}
	}
}
