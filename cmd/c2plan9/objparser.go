// Copyright (c) 2026 Lemon4ksan All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"debug/elf"
	"debug/macho"
	"debug/pe"
	"fmt"
	"os"
	"sort"
)

// Symbol represents an extracted function symbol with its byte slice.
type Symbol struct {
	Name    string
	Offset  uint64
	Size    uint64
	Code    []byte
	IsLocal bool
}

// ParsedObject contains the extracted machine code and symbol metadata.
type ParsedObject struct {
	Arch      string
	Symbols   []Symbol
	TextBytes []byte
	ROData    []byte
}

// ParseObject parses a binary object file (.o) across ELF, PE, or Mach-O formats.
func ParseObject(path string) (*ParsedObject, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read object file: %w", err)
	}

	// Try ELF first (standard output when targeting Linux/SysV)
	if obj, err := parseELF(data); err == nil {
		return obj, nil
	}

	// Try PE (Windows native)
	if obj, err := parsePE(data); err == nil {
		return obj, nil
	}

	// Try Mach-O (macOS native)
	if obj, err := parseMachO(data); err == nil {
		return obj, nil
	}

	return nil, fmt.Errorf("unsupported object file format: %s", path)
}

func parseELF(data []byte) (*ParsedObject, error) {
	f, err := elf.NewFile(bytesReaderAt(data))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	textSec := f.Section(".text")
	if textSec == nil {
		return nil, fmt.Errorf("ELF object missing .text section")
	}

	textBytes, err := textSec.Data()
	if err != nil {
		return nil, fmt.Errorf("failed to read ELF .text data: %w", err)
	}

	var rodataBytes []byte
	if roSec := f.Section(".rodata"); roSec != nil {
		rodataBytes, _ = roSec.Data()
	}

	syms, err := f.Symbols()
	if err != nil {
		return nil, fmt.Errorf("failed to read ELF symbols: %w", err)
	}

	var funcSyms []Symbol
	for _, sym := range syms {
		if sym.Section == elf.SHN_UNDEF {
			continue
		}
		// Function or STT_NOTYPE symbols with non-zero size or in .text
		if int(sym.Section) < len(f.Sections) && f.Sections[sym.Section].Name == ".text" {
			if sym.Size > 0 {
				end := sym.Value + sym.Size
				if end <= uint64(len(textBytes)) {
					funcSyms = append(funcSyms, Symbol{
						Name:    sym.Name,
						Offset:  sym.Value,
						Size:    sym.Size,
						Code:    textBytes[sym.Value:end],
						IsLocal: elf.ST_BIND(sym.Info) == elf.STB_LOCAL,
					})
				}
			}
		}
	}

	// If no function sizes were explicitly stored in ELF symbols, deduce by sort offsets
	if len(funcSyms) == 0 {
		var textSymbols []elf.Symbol
		for _, sym := range syms {
			if int(sym.Section) < len(f.Sections) && f.Sections[sym.Section].Name == ".text" && sym.Name != "" {
				textSymbols = append(textSymbols, sym)
			}
		}
		sort.Slice(textSymbols, func(i, j int) bool {
			return textSymbols[i].Value < textSymbols[j].Value
		})

		for i, sym := range textSymbols {
			start := sym.Value
			var end uint64
			if i+1 < len(textSymbols) {
				end = textSymbols[i+1].Value
			} else {
				end = uint64(len(textBytes))
			}
			if end > start && end <= uint64(len(textBytes)) {
				funcSyms = append(funcSyms, Symbol{
					Name:    sym.Name,
					Offset:  start,
					Size:    end - start,
					Code:    textBytes[start:end],
					IsLocal: elf.ST_BIND(sym.Info) == elf.STB_LOCAL,
				})
			}
		}
	}

	arch := "amd64"
	if f.Machine == elf.EM_AARCH64 {
		arch = "arm64"
	}

	return &ParsedObject{
		Arch:      arch,
		Symbols:   funcSyms,
		TextBytes: textBytes,
		ROData:    rodataBytes,
	}, nil
}

func parsePE(data []byte) (*ParsedObject, error) {
	f, err := pe.NewFile(bytesReaderAt(data))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	textSec := f.Section(".text")
	if textSec == nil {
		return nil, fmt.Errorf("PE object missing .text section")
	}

	textBytes, err := textSec.Data()
	if err != nil {
		return nil, fmt.Errorf("failed to read PE .text data: %w", err)
	}

	var funcSyms []Symbol
	var peSymbols []*pe.Symbol
	for _, sym := range f.Symbols {
		if sym.SectionNumber > 0 && int(sym.SectionNumber) <= len(f.Sections) {
			if f.Sections[sym.SectionNumber-1].Name == ".text" && sym.Name != "" {
				peSymbols = append(peSymbols, sym)
			}
		}
	}

	sort.Slice(peSymbols, func(i, j int) bool {
		return peSymbols[i].Value < peSymbols[j].Value
	})

	for i, sym := range peSymbols {
		start := uint64(sym.Value)
		var end uint64
		if i+1 < len(peSymbols) {
			end = uint64(peSymbols[i+1].Value)
		} else {
			end = uint64(len(textBytes))
		}

		if end > start && end <= uint64(len(textBytes)) {
			funcSyms = append(funcSyms, Symbol{
				Name:   sym.Name,
				Offset: start,
				Size:   end - start,
				Code:   textBytes[start:end],
			})
		}
	}

	arch := "amd64"
	if f.Machine == pe.IMAGE_FILE_MACHINE_ARM64 {
		arch = "arm64"
	}

	return &ParsedObject{
		Arch:      arch,
		Symbols:   funcSyms,
		TextBytes: textBytes,
	}, nil
}

func parseMachO(data []byte) (*ParsedObject, error) {
	f, err := macho.NewFile(bytesReaderAt(data))
	if err != nil {
		return nil, err
	}
	defer f.Close()

	textSec := f.Section("__text")
	if textSec == nil {
		return nil, fmt.Errorf("Mach-O object missing __text section")
	}

	textBytes, err := textSec.Data()
	if err != nil {
		return nil, fmt.Errorf("failed to read Mach-O __text data: %w", err)
	}

	arch := "amd64"
	if f.Cpu == macho.CpuArm64 {
		arch = "arm64"
	}

	var funcSyms []Symbol
	if f.Symtab != nil {
		for _, sym := range f.Symtab.Syms {
			if sym.Sect > 0 && sym.Name != "" {
				funcSyms = append(funcSyms, Symbol{
					Name:   sym.Name,
					Offset: sym.Value,
					Size:   uint64(len(textBytes)) - sym.Value,
					Code:   textBytes[sym.Value:],
				})
			}
		}
	}

	return &ParsedObject{
		Arch:      arch,
		Symbols:   funcSyms,
		TextBytes: textBytes,
	}, nil
}

type readerAtBytes []byte

func (r readerAtBytes) ReadAt(p []byte, off int64) (n int, err error) {
	if off >= int64(len(r)) {
		return 0, fmt.Errorf("offset out of bounds")
	}
	n = copy(p, r[off:])
	return n, nil
}

func bytesReaderAt(b []byte) readerAtBytes {
	return readerAtBytes(b)
}
