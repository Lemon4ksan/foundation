// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package uuid

//go:generate c2plan9 -c ../../csrc/uuid.c -o uuid_amd64.s -stub uuid_amd64.go -pkg uuid
