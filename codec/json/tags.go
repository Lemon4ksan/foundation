// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package json

import (
	"reflect"
	"strings"
)

// tagOptions contains parsed JSON struct tag options.
type tagOptions struct {
	name      string
	quoted    bool
	omitEmpty bool
	ignored   bool
}

// parseTag parses the `json` struct tag for a struct field.
func parseTag(field reflect.StructField) tagOptions {
	tag := field.Tag.Get("json")
	if tag == "-" {
		return tagOptions{ignored: true}
	}

	if tag == "" {
		if field.Anonymous {
			return tagOptions{name: ""}
		}
		return tagOptions{name: field.Name}
	}

	parts := strings.Split(tag, ",")
	name := parts[0]
	if name == "" && !field.Anonymous {
		name = field.Name
	}

	opts := tagOptions{name: name}
	for i := 1; i < len(parts); i++ {
		switch parts[i] {
		case "omitempty":
			opts.omitEmpty = true
		case "string":
			opts.quoted = true
		}
	}

	return opts
}
