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

package main

import (
	"fmt"
	flag "gopkg.in/docker/docker.v1/pkg/mflag"
	"os"
)

var (
	flagHelp           = flag.Bool([]string{"h", "-help"}, false, "Print this help message.")
	flagVersion        = flag.Bool([]string{"v", "-version"}, false, "Print version information and quit.")
	flagReport         = flag.Bool([]string{"r", "-report"}, false, "Print detailed version report and quit.")
	flagProfile        = flag.Bool([]string{"ps", "-profilespeed"}, false, "Write msg/sec measurements to log.")
	flagLoglevel       = flag.Int([]string{"ll", "-loglevel"}, 0, "Set the loglevel [0-3]. Higher levels produce more messages.")
	flagNumCPU         = flag.Int([]string{"n", "-numcpu"}, 0, "Number of CPUs to use. Set 0 for all CPUs.")
	flagMetricsPort    = flag.Int([]string{"m", "-metrics"}, 0, "Port to use for metric queries. Set 0 to disable.")
	flagConfigFile     = flag.String([]string{"c", "-config"}, "", "Use a given configuration file.")
	flagTestConfigFile = flag.String([]string{"tc", "-testconfig"}, "", "Test a given configuration file and exit.")
	flagCPUProfile     = flag.String([]string{"pc", "-profilecpu"}, "", "Write CPU profiler results to a given file.")
	flagMemProfile     = flag.String([]string{"pm", "-profilemem"}, "", "Write heap profile results to a given file.")
	flagTrace          = flag.String([]string{"tr", "-trace"}, "", "Write trace results to a given file.")
	flagPidFile        = flag.String([]string{"p", "-pidfile"}, "", "Write the process id into a given file.")
)

func init() {
	flag.Usage = func() {
		fmt.Println("Usage: xPipeline [OPTIONS]\n\nxPipeline - Make pipes,Link the world!.\n\nOptions:")
		flag.CommandLine.SetOutput(os.Stdout)
		flag.PrintDefaults()
		fmt.Print("\n")
	}
}

func parseFlags() {
	flag.Parse()
}

func printFlags() {
	flag.Usage()
}
