package sql

import (
	"fmt"
	"reflect"
	"time"

	"gnd.la/encoding/codec"
	"gnd.la/encoding/pipe"
	"gnd.la/util/structs"

	"gopkgs.com/pool.v1"
)

var (
	scannerPool = pool.New(64)
)

type scanner struct {
	Out     *reflect.Value
	Tag     *structs.Tag
	Nil     bool
	Backend Backend
}

// Always assume the type is right
func (s *scanner) Scan(src interface{}) error {
	switch x := src.(type) {
	case nil:
		// Assign zero to the type
		s.Nil = true
		s.Out.Set(reflect.Zero(s.Out.Type()))
		return nil
	case int64:
		return s.Backend.ScanInt(x, s.Out, s.Tag)
	case float64:
		return s.Backend.ScanFloat(x, s.Out, s.Tag)
	case bool:
		return s.Backend.ScanBool(x, s.Out, s.Tag)
	case []byte:
		s.Nil = len(x) == 0
		if c := codec.FromTag(s.Tag); c != nil {
			if p := pipe.FromTag(s.Tag); p != nil {
				var err error
				if x, err = p.Decode(x); err != nil {
					return err
				}
			}
			addr := s.Out.Addr()
			return c.Decode(x, addr.Interface())
		}

		return s.Backend.ScanByteSlice(x, s.Out, s.Tag)
	case string:
		return s.Backend.ScanString(x, s.Out, s.Tag)
	case time.Time:
		return s.Backend.ScanTime(&x, s.Out, s.Tag)
	}
	return fmt.Errorf("can't scan value %v (%T)", src, src)
}

func newScanner(val *reflect.Value, t *structs.Tag, backend Backend) *scanner {
	if x := scannerPool.Get(); x != nil {
		s := x.(*scanner)
		s.Out = val
		s.Tag = t
		s.Nil = false
		s.Backend = backend
		return s
	}
	return &scanner{Out: val, Tag: t, Backend: backend}
}
