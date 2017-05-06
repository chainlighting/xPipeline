package core
import (
)

// Tidings is the interface definition for message split and process
type Tidings interface {
	Identify(data []byte) (msgLen int,startOffset int,padOffset int)
	Process(msg []byte,sequence uint64)
	DumpMetric()
}

