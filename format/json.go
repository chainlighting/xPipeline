// Copyright 2015-2017 trivago GmbH
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

package format

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/chainlighting/xPipeline/core"
	"github.com/chainlighting/xPipeline/core/log"
	"github.com/chainlighting/xPipeline/shared"
	"strconv"
	"sync"
	"time"
)

type jsonReaderState int

const (
	jsonReadArrayEnd    = jsonReaderState(iota)
	jsonReadObjectEnd   = jsonReaderState(iota)
	jsonReadObject      = jsonReaderState(iota)
	jsonReadKey         = jsonReaderState(iota)
	jsonReadValue       = jsonReaderState(iota)
	jsonReadArray       = jsonReaderState(iota)
	jsonReadArrayAppend = jsonReaderState(iota)
)

// JSON formatter plugin
// JSON is a formatter that passes a message encapsulated as JSON in the form
// {"message":"..."}. The actual message is formatted by a nested formatter and
// HTML escaped.
// Configuration example
//
//  - "stream.Broadcast":
//    Formatter: "format.JSON"
//    JSONStartState: "findKey"
//    JSONDirectives:
//	    - 'findKey :":  key     ::'
//	    - 'findKey :}:          : pop  : end'
//	    - 'key     :":  findVal :      : key'
//	    - 'findVal :\:: value   ::'
//
// JSONStartState defines the initial parser state when parsing a message.
// By default this is set to "" which will fall back to the first state used in
// the JSONDirectives array.
//
// JSONTimestampRead defines the go timestamp format expected from fields that
// are parsed as "dat". When JSONUnixTimestampRead is not set, this is set to
// "20060102150405" by default.
//
// JSONUnixTimestampRead defines the unix timestamp format expected from fields that
// are parsed as "dat". May be "s", "ms", or "ns", and only accepts integer values.
// When JSONTimestampRead is set, this is ignored.
//
// JSONTimestampWrite defines the go timestamp format that "dat" fields will be
// converted to. By default this is set to "2006-01-02 15:04:05 MST".
//
// JSONDirectives defines an array of parser directives.
// This setting is mandatory and has no default value.
// Each string must be of the following format: "State:Token:NextState:Flags:Function".
// Spaces will be stripped from all fields but Token. If a fields requires a
// colon it has to be escaped with a backslash. Other escape characters
// supported are \n, \r and \t.
//
// Flags (JSONDirectives) can be a comma separated set of the following flags.
//  * continue -> Prepend the token to the next match.
//  * append   -> Append the token to the current match and continue reading.
//  * include  -> Append the token to the current match.
//  * push     -> Push the current state to the stack.
//  * pop      -> Pop the stack and use the returned state if possible.
//
// Function (JSONDirectives) can hold one of the following names.
//  * key     -> Write the current match as a key.
//  * val     -> Write the current match as a value without quotes.
//  * esc     -> Write the current match as a escaped string value.
//  * dat     -> Write the current match as a timestamp value.
//  * arr     -> Start a new array.
//  * obj     -> Start a new object.
//  * end     -> Close an array or object.
//  * arr+val -> arr followed by val.
//  * arr+esc -> arr followed by esc.
//  * arr+dat -> arr followed by dat.
//  * val+end -> val followed by end.
//  * esc+end -> esc followed by end.
//  * dat+end -> dat followed by end.
//
// Rules for storage (JSONDirectives): if a value is written without a previous key write, a key will be auto
// generated from the current parser state name. This does not happen when
// inside an array.
// If key is written without a previous value write, a null value will be
// written. This does not happen after an object has been started.
// A key write inside an array will cause the array to be closed. If the array
// is nested, all arrays will be closed.
type JSON struct {
	message   *bytes.Buffer
	parser    shared.TransitionParser
	state     jsonReaderState
	stack     []jsonReaderState
	parseLock *sync.Mutex
	initState string
	timeRead  string
	timeWrite string
	timeParse func(string, string) (time.Time, error)
}

func init() {
	shared.TypeRegistry.Register(JSON{})
}

func parseUnix(layout, value string) (time.Time, error) {
	s, ns := int64(0), int64(0)
	switch layout {
	case "s":
		valueInt, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		s = valueInt
	case "ms":
		valueInt, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		ns = valueInt * int64(time.Millisecond)
	case "ns":
		valueInt, err := strconv.ParseInt(value, 10, 64)
		if err != nil {
			return time.Time{}, err
		}
		ns = valueInt
	}
	return time.Unix(s, ns), nil
}

// Configure initializes this formatter with values from a plugin config.
func (format *JSON) Configure(conf core.PluginConfig) error {
	format.parser = shared.NewTransitionParser()
	format.state = jsonReadObject
	format.initState = conf.GetString("JSONStartState", "")
	format.timeRead = conf.GetString("JSONTimestampRead", "")
	format.timeWrite = conf.GetString("JSONTimestampWrite", "2006-01-02 15:04:05 MST")
	format.timeParse = time.Parse
	format.parseLock = new(sync.Mutex)

	unixRead := conf.GetString("JSONUnixTimestampRead", "")
	if format.timeRead == "" {
		if unixRead == "" {
			// Use default when neither are specified
			format.timeRead = "20060102150405"
		} else {
			format.timeRead = unixRead
			format.timeParse = parseUnix
		}
	} else if unixRead != "" {
		Log.Warning.Print("Cannot use both JSONTimestampRead and JSONUnixTimestampRead, defaulting to JSONTimestampRead")
	}

	if !conf.HasValue("JSONDirectives") {
		Log.Warning.Print("JSON formatter has no JSONDirectives setting")
		return nil // ### return, no directives ###
	}

	directiveStrings := conf.GetStringArray("JSONDirectives", []string{})
	if len(directiveStrings) == 0 {
		Log.Warning.Print("JSON formatter has no directives")
		return nil // ### return, no directives ###
	}

	// Parse directives

	parserFunctions := make(map[string]shared.ParsedFunc)
	parserFunctions["key"] = format.readKey
	parserFunctions["val"] = format.readValue
	parserFunctions["esc"] = format.readEscaped
	parserFunctions["dat"] = format.readDate
	parserFunctions["arr"] = format.readArray
	parserFunctions["obj"] = format.readObject
	parserFunctions["end"] = format.readEnd
	parserFunctions["arr+val"] = format.readArrayValue
	parserFunctions["arr+esc"] = format.readArrayEscaped
	parserFunctions["val+end"] = format.readValueEnd
	parserFunctions["esc+end"] = format.readEscapedEnd
	parserFunctions["dat+end"] = format.readDateEnd

	directives := []shared.TransitionDirective{}
	for _, dirString := range directiveStrings {
		directive, err := shared.ParseTransitionDirective(dirString, parserFunctions)
		if err != nil {
			return fmt.Errorf("%s: %s", err.Error(), dirString) // ### return, malformed directive ###
		}
		if format.initState == "" {
			format.initState = directive.State
		}
		directives = append(directives, directive)
	}

	format.parser.AddDirectives(directives)

	// Validate initstate
	initStateValid := false
	for i := 0; i < len(directives) && !initStateValid; i++ {
		initStateValid = directives[i].State == format.initState
	}
	if !initStateValid {
		return fmt.Errorf("JSONStartState does not exist in directives")
	}

	for _, dir := range directives {
		nextStateValid := false
		for i := 0; i < len(directives) && !nextStateValid; i++ {
			nextStateValid = dir.NextState == directives[i].State
		}
		if !nextStateValid {
			Log.Warning.Printf("State \"%s\" has a transition to \"%s\" which does not exist in directives", dir.State, dir.NextState)
		}
	}

	return nil
}

func (format *JSON) writeKey(key []byte) {
	// Make sure we are not in an array anymore
	for format.state == jsonReadArray || format.state == jsonReadArrayAppend {
		format.readEnd(nil, 0)
	}

	// If no value was written, write null
	if format.state > jsonReadKey {
		format.message.WriteString("null")
	}

	// Prepend a comma except after an object has started
	if format.state != jsonReadObject {
		format.message.WriteByte(',')
	}

	format.message.WriteByte('"')
	format.message.Write(key)
	format.message.WriteString(`":`)
}

func (format *JSON) readKey(data []byte, state shared.ParserStateID) {
	format.writeKey(data)
	format.state = jsonReadValue
}

func (format *JSON) readValue(data []byte, state shared.ParserStateID) {
	trimmedData := bytes.TrimSpace(data)
	if len(trimmedData) == 0 {
		switch format.state {
		default:
			format.state = jsonReadKey
		case jsonReadArray, jsonReadArrayAppend:
		}
		return
	}

	switch format.state {
	default:
		format.writeKey([]byte(format.parser.GetStateName(state)))
		fallthrough

	case jsonReadValue:
		format.message.Write(trimmedData)
		format.state = jsonReadKey

	case jsonReadArray:
		format.message.Write(trimmedData)
		format.state = jsonReadArrayAppend

	case jsonReadArrayAppend:
		format.message.WriteByte(',')
		format.message.Write(trimmedData)
	}
}

func (format *JSON) readEscaped(data []byte, state shared.ParserStateID) {
	trimmedData := bytes.TrimSpace(data)
	if len(trimmedData) == 0 {
		switch format.state {
		default:
			format.state = jsonReadKey
		case jsonReadArrayAppend, jsonReadArray:
		}
		return
	}

	encodedData, _ := json.Marshal(string(trimmedData))
	switch format.state {
	default:
		format.writeKey([]byte(format.parser.GetStateName(state)))
		fallthrough

	case jsonReadValue:
		format.message.Write(encodedData)
		format.state = jsonReadKey

	case jsonReadArray:
		format.message.Write(encodedData)
		format.state = jsonReadArrayAppend

	case jsonReadArrayAppend:
		format.message.WriteString(`,`)
		format.message.Write(encodedData)
	}
}

func (format *JSON) readDate(data []byte, state shared.ParserStateID) {
	date, _ := format.timeParse(format.timeRead, string(bytes.TrimSpace(data)))
	formattedDate := date.Format(format.timeWrite)
	format.readEscaped([]byte(formattedDate), state)
}

func (format *JSON) readValueEnd(data []byte, state shared.ParserStateID) {
	formatState := format.state
	format.readValue(data, state)
	format.state = formatState
	format.readEnd(data, state)
}

func (format *JSON) readEscapedEnd(data []byte, state shared.ParserStateID) {
	formatState := format.state
	format.readEscaped(data, state)
	format.state = formatState
	format.readEnd(data, state)
}

func (format *JSON) readDateEnd(data []byte, state shared.ParserStateID) {
	formatState := format.state
	format.readDate(data, state)
	format.state = formatState
	format.readEnd(data, state)
}

func (format *JSON) readArrayValue(data []byte, state shared.ParserStateID) {
	format.readArray(data, state)
	format.readValue(data, state)
}

func (format *JSON) readArrayEscaped(data []byte, state shared.ParserStateID) {
	format.readArray(data, state)
	format.readEscaped(data, state)
}

func (format *JSON) readArrayDate(data []byte, state shared.ParserStateID) {
	format.readArray(data, state)
	format.readDate(data, state)
}

func (format *JSON) readArray(data []byte, state shared.ParserStateID) {
	switch format.state {
	default:
		format.writeKey([]byte(format.parser.GetStateName(state)))
		fallthrough

	case jsonReadValue, jsonReadArray, jsonReadObject:
		format.message.WriteByte('[')

	case jsonReadArrayAppend:
		format.message.WriteString(",[")
	}
	format.stack = append(format.stack, format.state)
	format.state = jsonReadArray
}

func (format *JSON) readObject(data []byte, state shared.ParserStateID) {
	switch format.state {
	default:
		format.writeKey([]byte(format.parser.GetStateName(state)))
		fallthrough

	case jsonReadValue, jsonReadObject, jsonReadArray:
		format.message.WriteByte('{')

	case jsonReadArrayAppend:
		format.message.WriteString(",{")
	}
	format.stack = append(format.stack, format.state)
	format.state = jsonReadObject
}

func (format *JSON) readEnd(data []byte, state shared.ParserStateID) {
	stackSize := len(format.stack)

	if stackSize > 0 {
		switch format.state {
		case jsonReadArray, jsonReadArrayAppend:
			format.message.WriteByte(']')
		default:
			format.message.WriteByte('}')
		}
	}

	if stackSize > 1 {
		format.state = format.stack[stackSize-1]
		format.stack = format.stack[:stackSize-1] // Pop the stack
		if format.state == jsonReadArray {
			format.state = jsonReadArrayAppend // just finished the first entry
		}
	} else {
		format.stack = format.stack[:0] // Clear the stack
		format.state = jsonReadKey
	}
}

// Format parses the incoming message and generates JSON from it.
// This function is mutex locked.
func (format *JSON) Format(msg core.Message) ([]byte, core.MessageStreamID) {
	// The internal state is not threadsafe so we need to lock here
	format.parseLock.Lock()
	defer format.parseLock.Unlock()

	format.message = bytes.NewBuffer(nil)
	format.state = jsonReadObject

	format.message.WriteString("{")
	remains, state := format.parser.Parse(msg.Data, format.initState)

	// Write remains as string value
	if remains != nil {
		format.readEscaped(remains, state)
	}

	// Close any open tags
	if format.message.Len() > 1 {
		for format.state == jsonReadArray || format.state == jsonReadArrayAppend || format.state == jsonReadObject {
			format.readEnd(nil, 0)
		}
	}

	format.message.WriteString("}\n")
	return bytes.TrimSpace(format.message.Bytes()), msg.StreamID
}
