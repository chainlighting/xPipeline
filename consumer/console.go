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

package consumer

import (
	"github.com/chainlighting/xPipeline/core"
	"github.com/chainlighting/xPipeline/core/log"
	"github.com/chainlighting/xPipeline/shared"
	"io"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	consoleBufferGrowSize = 256
)

// Console consumer plugin
// This consumer reads from stdin. A message is generated after each newline
// character. When attached to a fuse, this consumer will stop accepting
// messages in case that fuse is burned.
// Configuration example
//
//  - "consumer.Console":
//    Console: "stdin"
//    Permissions: "0664"
//    ExitOnEOF: false
//
// Console defines the pipe to read from. This can be "stdin" or the name
// of a named pipe that is created if not existing. The default is "stdin"
//
// Permissions accepts an octal number string that contains the unix file
// permissions used when creating a named pipe.
// By default this is set to "0664".
//
// ExitOnEOF can be set to true to trigger an exit signal if StdIn is closed
// (e.g. when a pipe is closed). This is set to false by default.
type Console struct {
	core.ConsumerBase
	autoexit bool
	pipe     *os.File
	pipeName string
	pipePerm uint32
}

func init() {
	shared.TypeRegistry.Register(Console{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Console) Configure(conf core.PluginConfig) error {
	cons.autoexit = conf.GetBool("ExitOnEOF", false)
	inputConsole := conf.GetString("Console", "stdin")

	switch strings.ToLower(inputConsole) {
	case "stdin":
		cons.pipe = os.Stdin
		cons.pipeName = "stdin"
	default:
		cons.pipe = nil
		cons.pipeName = inputConsole

		if perm, err := strconv.ParseInt(conf.GetString("Permissions", "0664"), 8, 32); err != nil {
			Log.Error.Printf("Error parsing named pipe permissions: %s", err)
		} else {
			cons.pipePerm = uint32(perm)
		}
	}

	return cons.ConsumerBase.Configure(conf)
}

func (cons *Console) readPipe() {
	if cons.pipe == nil {
		var err error
		if cons.pipe, err = shared.OpenNamedPipe(cons.pipeName, cons.pipePerm); err != nil {
			Log.Error.Print(err)
			time.AfterFunc(3*time.Second, cons.readPipe)
			return // ### return, try again ###
		}

		defer cons.pipe.Close()
	}

	buffer := shared.NewBufferedReader(consoleBufferGrowSize, 0, 0, "\n",nil)
	for cons.IsActive() {
		err := buffer.ReadAll(cons.pipe, cons.Enqueue)
		cons.WaitOnFuse()
		switch err {
		case io.EOF:
			if cons.autoexit {
				// TODO: Hack
				Log.Debug.Print("Console triggered exit.")
				proc, _ := os.FindProcess(os.Getpid())
				proc.Signal(os.Interrupt)
			}

		case nil:
			// ignore
		default:
			Log.Error.Print("Error reading pipe: ", err)
		}
	}
}

// Consume listens to stdin.
func (cons *Console) Consume(workers *sync.WaitGroup) {
	go cons.readPipe()
	cons.ControlLoop()
}
