// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package bytesconv

//go:generate c2plan9 -c ../../csrc/casing.c -o casing_amd64.s -stub casing_amd64.go -pkg bytesconv
//go:generate c2plan9 -c ../../csrc/base64.c -o base64_amd64.s -stub base64_amd64.go -pkg bytesconv
