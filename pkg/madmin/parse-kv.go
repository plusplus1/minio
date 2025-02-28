/*
 * MinIO Cloud Storage, (C) 2019 MinIO, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 *
 */

package madmin

import (
	"bufio"
	"bytes"
	"fmt"
	"strings"
	"unicode"
)

// KVS each sub-system key, value
type KVS map[string]string

// Targets sub-system targets
type Targets map[string]map[string]KVS

const (
	stateKey   = "state"
	commentKey = "comment"

	stateOn  = "on"
	stateOff = "off"
)

func (kvs KVS) String() string {
	var s strings.Builder
	for k, v := range kvs {
		// Do not need to print if state is on
		if k == stateKey && v == stateOn {
			continue
		}
		if k == commentKey && v == "" {
			continue
		}
		s.WriteString(k)
		s.WriteString(KvSeparator)
		spc := hasSpace(v)
		if spc {
			s.WriteString(KvDoubleQuote)
		}
		s.WriteString(v)
		if spc {
			s.WriteString(KvDoubleQuote)
		}
		s.WriteString(KvSpaceSeparator)
	}
	return s.String()
}

// Count - returns total numbers of target
func (t Targets) Count() int {
	var count int
	for _, targetKV := range t {
		for range targetKV {
			count++
		}
	}
	return count
}

func hasSpace(s string) bool {
	for _, r := range s {
		if unicode.IsSpace(r) {
			return true
		}
	}
	return false
}

func (t Targets) String() string {
	var s strings.Builder
	count := t.Count()
	for subSys, targetKV := range t {
		for target, kv := range targetKV {
			count--
			s.WriteString(subSys)
			if target != Default {
				s.WriteString(SubSystemSeparator)
				s.WriteString(target)
			}
			s.WriteString(KvSpaceSeparator)
			s.WriteString(kv.String())
			if (len(t) > 1 || len(targetKV) > 1) && count > 0 {
				s.WriteString(KvNewline)
			}
		}
	}
	return s.String()
}

// Constant separators
const (
	SubSystemSeparator = `:`
	KvSeparator        = `=`
	KvSpaceSeparator   = ` `
	KvNewline          = "\n"
	KvDoubleQuote      = `"`
	KvSingleQuote      = `'`

	Default = `_`
)

// This function is needed, to trim off single or double quotes, creeping into the values.
func sanitizeValue(v string) string {
	v = strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(v), KvDoubleQuote), KvDoubleQuote)
	return strings.TrimSuffix(strings.TrimPrefix(v, KvSingleQuote), KvSingleQuote)
}

func convertTargets(s string, targets Targets) error {
	inputs := strings.SplitN(s, KvSpaceSeparator, 2)
	if len(inputs) <= 1 {
		return fmt.Errorf("invalid number of arguments '%s'", s)
	}
	subSystemValue := strings.SplitN(inputs[0], SubSystemSeparator, 2)
	if len(subSystemValue) == 0 {
		return fmt.Errorf("invalid number of arguments %s", s)
	}
	var kvs = KVS{}
	var prevK string
	for _, v := range strings.Fields(inputs[1]) {
		kv := strings.SplitN(v, KvSeparator, 2)
		if len(kv) == 0 {
			continue
		}
		if len(kv) == 1 && prevK != "" {
			kvs[prevK] = strings.Join([]string{kvs[prevK], sanitizeValue(kv[0])}, KvSpaceSeparator)
			continue
		}
		if len(kv) == 1 {
			return fmt.Errorf("value for key '%s' cannot be empty", kv[0])
		}
		prevK = kv[0]
		kvs[kv[0]] = sanitizeValue(kv[1])
	}

	_, ok := targets[subSystemValue[0]]
	if !ok {
		targets[subSystemValue[0]] = map[string]KVS{}
	}
	if len(subSystemValue) == 2 {
		targets[subSystemValue[0]][subSystemValue[1]] = kvs
	} else {
		targets[subSystemValue[0]][Default] = kvs
	}
	return nil
}

// ParseSubSysTarget - parse sub-system target
func ParseSubSysTarget(buf []byte) (Targets, error) {
	targets := make(map[string]map[string]KVS)
	bio := bufio.NewScanner(bytes.NewReader(buf))
	for bio.Scan() {
		if err := convertTargets(bio.Text(), targets); err != nil {
			return nil, err
		}
	}
	if err := bio.Err(); err != nil {
		return nil, err
	}
	return targets, nil
}
