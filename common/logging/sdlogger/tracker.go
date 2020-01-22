// Copyright 2018 The LUCI Authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package sdlogger

import (
	"sync/atomic"
)

// SeverityTracker wraps LogEntryWriter and observes severity of messages there.
type SeverityTracker struct {
	Out LogEntryWriter

	debug int32
	info  int32
	warn  int32
	err   int32
}

// Write is part of LogEntryWriter interface.
func (s *SeverityTracker) Write(l *LogEntry) {
	s.Out.Write(l)

	var ptr *int32
	switch l.Severity {
	case DebugSeverity:
		ptr = &s.debug
	case InfoSeverity:
		ptr = &s.info
	case WarningSeverity:
		ptr = &s.warn
	case ErrorSeverity:
		ptr = &s.err
	default:
		return
	}

	if *ptr == 0 {
		atomic.StoreInt32(ptr, 1)
	}
}

// MaxSeverity returns maximum severity observed thus far or "".
func (s *SeverityTracker) MaxSeverity() Severity {
	switch {
	case atomic.LoadInt32(&s.err) == 1:
		return ErrorSeverity
	case atomic.LoadInt32(&s.warn) == 1:
		return WarningSeverity
	case atomic.LoadInt32(&s.info) == 1:
		return InfoSeverity
	case atomic.LoadInt32(&s.debug) == 1:
		return DebugSeverity
	default:
		return UnknownSeverity
	}
}
