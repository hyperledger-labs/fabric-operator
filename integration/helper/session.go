/*
 * Copyright contributors to the Hyperledger Fabric Operator project
 *
 * SPDX-License-Identifier: Apache-2.0
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at:
 *
 * 	  http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package helper

import (
	"fmt"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/onsi/ginkgo/v2"
	"github.com/onsi/gomega/gexec"
)

var (
	colorIndex uint
)

func AbsPath(wd string, script string) string {
	return filepath.Join(wd, script)
}

func GetCommand(command string, args ...string) *exec.Cmd {
	for _, arg := range args {
		command = command + " " + arg
	}
	// Ignoring this gosec issue as this is integration test code
	return exec.Command("bash", "-c", command) // #nosec
}

// StartSession executes a command session. This should be used to launch
// command line tools that are expected to run to completion.
func StartSession(cmd *exec.Cmd, name string) (*gexec.Session, error) {
	ansiColorCode := nextColor()
	fmt.Fprintf(
		ginkgo.GinkgoWriter,
		"\x1b[33m[d]\x1b[%s[%s]\x1b[0m starting %s %s with env var: %s\n",
		ansiColorCode,
		name,
		filepath.Base(cmd.Args[0]),
		strings.Join(cmd.Args[1:], " "),
		cmd.Env,
	)
	return gexec.Start(
		cmd,
		gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[32m[o]\x1b[%s[%s]\x1b[0m ", ansiColorCode, name),
			ginkgo.GinkgoWriter,
		),
		gexec.NewPrefixedWriter(
			fmt.Sprintf("\x1b[91m[e]\x1b[%s[%s]\x1b[0m ", ansiColorCode, name),
			ginkgo.GinkgoWriter,
		),
	)
}

func nextColor() string {
	color := colorIndex%14 + 31
	if color > 37 {
		color = color + 90 - 37
	}

	colorIndex++
	return fmt.Sprintf("%dm", color)
}
