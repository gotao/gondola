package runtimeutil

import (
	"debug/macho"
	"fmt"
	"io"
)

type file struct {
	*macho.File
}

func (f *file) section(name string) ([]byte, error) {
	s := f.Section(name)
	if s == nil {
		return nil, fmt.Errorf("no section name %q", name)
	}
	return s.Data()
}

func (f *file) Symtab() ([]byte, error) {
	return f.section("__gosymtab")
}

func (f *file) Pclntab() ([]byte, error) {
	return f.section("__gopclntab")
}

func (f *file) TextAddr() uint64 {
	return f.Section("__text").Addr
}

func openDebugFile(r io.ReaderAt) (debugFile, error) {
	f, err := macho.NewFile(r)
	if err != nil {
		return nil, err
	}
	return &file{f}, nil
}
