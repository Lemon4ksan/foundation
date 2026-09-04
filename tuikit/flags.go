// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package tuikit

import (
	"github.com/lemon4ksan/foundation/argkit"
)

// StringSliceFlag implements [flag.Value] for multi-valued command line arguments.
//
// Deprecated: Use [argkit.StringSliceFlag] directly.
type StringSliceFlag = argkit.StringSliceFlag

// NormalizeArgs stitches back arguments that were fragmented by shell tokenizers.
//
// Deprecated: Use [argkit.NormalizeArgs] directly.
var NormalizeArgs = argkit.NormalizeArgs

// ParseInterspersedFlags parses a FlagSet correctly with full POSIX semantics.
//
// Deprecated: Use [argkit.ParseInterspersedFlags] directly.
var ParseInterspersedFlags = argkit.ParseInterspersedFlags

// StringVar binds a string flag with optional short alias.
//
// Deprecated: Use [argkit.StringVar] directly.
var StringVar = argkit.StringVar

// BoolVar binds a boolean flag with optional short alias.
//
// Deprecated: Use [argkit.BoolVar] directly.
var BoolVar = argkit.BoolVar

// IntVar binds an integer flag with optional short alias.
//
// Deprecated: Use [argkit.IntVar] directly.
var IntVar = argkit.IntVar

// Int64Var binds an int64 flag with optional short alias.
//
// Deprecated: Use [argkit.Int64Var] directly.
var Int64Var = argkit.Int64Var

// Float64Var binds a float64 flag with optional short alias.
//
// Deprecated: Use [argkit.Float64Var] directly.
var Float64Var = argkit.Float64Var

// DurationVar binds a time.Duration flag with optional short alias.
//
// Deprecated: Use [argkit.DurationVar] directly.
var DurationVar = argkit.DurationVar
