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

package consumer

import (
	"github.com/chainlighting/xPipeline/core"
	"github.com/chainlighting/xPipeline/core/log"
	"github.com/chainlighting/xPipeline/shared"
	"io"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

const (
	fileBufferGrowSize = 1024
)

type fileState int32

const (
	fileStateOpen = fileState(iota)
	fileStateRead = fileState(iota)
	fileStateDone = fileState(iota)
)


//  - "consumer.Files":
//      "Mode": "local"
//      "concurrency": 3
//      "Path": "/var/dir/file"
//      "Remain": true


// files process start
const (
	filesMethodLocal     = "local"
	filesMethodFtpClient = "ftpclient"
	filesMethodFtpServer = "ftpserver"
)

// msg parser special
const (
	msgSplitAsDefault    = "default"
)

type MsgSplitFunc func(data []byte, dataSize int) (int,int)
type FilesGroupId uint64

type FileInstance struct {
	file           *os.File
	pathfile       string
	state          fileState
	curRetryTimes  int
}

type FilesGroup struct {
	method         string
	concurrency    int
	username       string
	passwd         string
	path           string
	remain         bool
	// delay x seconds to retry,0 is no delay
	retryDelay     int
	// the times to retry,0 is alway retry
	retryTimes     int
	// special msg parse func	
	splitFunc      MsgSplitFunc
	// pathfile ==> FileInstance
	files          map[string]*FileInstance
	delayFiles     []*FileInstance
	processingFiles []*FileInstance
}

type Files struct {
	core.ConsumerBase
	groups         FilesGroup
}

func init() {
	shared.TypeRegistry.Register(Files{})
}

// Configure initializes this consumer with values from a plugin config.
func (cons *Files) Configure(conf core.PluginConfig) error {
	err := cons.ConsumerBase.Configure(conf)
	if err != nil {
		return err
	}

	cons.groups = FilesGroup{}
	cons.groups.method = conf.GetString("Method",filesMethodLocal)
	cons.groups.concurrency = conf.GetInt("Concurrency",3)
	cons.groups.username = conf.GetString("Username","xPipeline")
	cons.groups.passwd = conf.GetString("Passwd","xPipeline")
	cons.groups.path = conf.GetString("Path","/var/log/system.log")
	cons.groups.remain = conf.GetBool("Remain",true)
	cons.groups.retryDelay = conf.GetInt("RetryDelay",10)
	cons.groups.retryTimes = conf.GetInt("RetryTimes",3)
	cons.groups.files = make(map[string]*FileInstance)
	cons.groups.delayFiles = make([]*FileInstance,32)
	cons.groups.processingFiles = make([]*FileInstance,32)

	// set spliter function
	spliter := strings.ToLower(conf.GetString("SpliterAs",msgSplitAsDefault))
	switch spliter {
	case msgSplitAsDefault:
		cons.groups.splitFunc = nil
	default:
		cons.groups.splitFunc = nil
	}

	return nil
}

func (cons *Files) start() {
	// enum all files by path and cached in cons.groups.files
	// get some file to process by cons.groups.concurrency from cons.groups.files
	// wait subroutine ack,maybe need to take incomplete files to delayFiles
}

// Consume listens to stdin.
func (cons *Files) Consume(workers *sync.WaitGroup) {
	go shared.DontPanic(func() {
		cons.AddMainWorker(workers)
		cons.start()
	})

	cons.ControlLoop()
}

