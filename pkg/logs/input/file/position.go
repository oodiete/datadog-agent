// Unless explicitly stated otherwise all files in this repository are licensed
// under the Apache License Version 2.0.
// This product includes software developed at Datadog (https://www.datadoghq.com/).
// Copyright 2016-2020 Datadog, Inc.

package file

import (
	"io"
	"strconv"

	"github.com/DataDog/datadog-agent/pkg/logs/auditor"
)

// TailingMode type
type TailingMode uint8

// Tailing Modes
const (
	ForceBeginning = iota
	ForceEnd
	Beginning
	End
)

// Tailing mode string representation
const (
	ForceBeginningStr = "forceBeginning"
	ForceEndStr       = "forceEnd"
	BeginningStr      = "beginning"
	EndStr            = "end"
)

var tailingModeToStringRepresentations = map[TailingMode]string{
	ForceBeginning: ForceBeginningStr,
	ForceEnd:       ForceEndStr,
	Beginning:      BeginningStr,
	End:            EndStr,
}

// TailingModeFromString parses a string and returns a corresponding tailing mode, default to End if not found
func TailingModeFromString(mode string) (TailingMode, bool) {
	for m, mStr := range tailingModeToStringRepresentations {
		if mStr == mode {
			return m, true
		}
	}
	return End, false
}

// TailingModeToString returns seelog string representation for a specified tailing mode. Returns "" for invalid tailing mode.
func (mode TailingMode) String() string {
	modeStr, ok := tailingModeToStringRepresentations[mode]
	if ok {
		return modeStr
	}
	return ""
}

// Position returns the position from where logs should be collected.
func Position(registry auditor.Registry, identifier string, m TailingMode) (int64, int, error) {
	var offset int64
	var whence int
	var err error

	value := registry.GetOffset(identifier)

	switch {
	case m == ForceBeginning:
		offset, whence = 0, io.SeekStart
	case m == ForceEnd:
		offset, whence = 0, io.SeekEnd
	case value != "":
		// an offset was registered, tailing mode is not forced, tail from the offset
		whence = io.SeekStart
		offset, err = strconv.ParseInt(value, 10, 64)
		if err != nil {
			offset = 0
			if m == End {
				whence = io.SeekEnd
			}
			if m == Beginning {
				whence = io.SeekStart
			}
		}
	case m == Beginning:
		offset, whence = 0, io.SeekStart
	case m == End:
		fallthrough
	default:
		offset, whence = 0, io.SeekEnd
	}
	return offset, whence, err
}
