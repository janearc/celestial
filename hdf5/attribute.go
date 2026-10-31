package hdf5

import (
	"bytes"
	"fmt"
)

// versionOneAttribute is the version of the Attribute message that is
// read. Defined by IV.A.2.m; versions 2 and 3 do not pad their fields and
// may share their datatype and dataspace.
const versionOneAttribute = 1

// attributeAlign is the multiple of bytes the name, the datatype and the
// dataspace of a version 1 attribute are each padded to. Defined by
// IV.A.2.m.
const attributeAlign = 8

// nameEnd is the byte that ends an attribute's name. Defined by IV.A.2.m:
// a null.
const nameEnd = 0

// notFound is what bytes.IndexByte gives for a byte it does not find.
// Defined by the bytes package.
const notFound = -1

// named describes an attribute of a group or a dataset.
type named struct {
	// name is the attribute's name.
	name string
	// shape is the size of each dimension; a scalar has none.
	shape []uint64
	// kind is the type of one element.
	kind datatype
}

// attribute is an attribute read: its description and its bytes, which
// are in the message, little-endian.
type attribute struct {
	named
	raw []byte
}

// readAttribute parses an Attribute message. IV.A.2.m, version 1. It
// refuses a message too short for its parts and a version not read, and
// then gives none.
func readAttribute(msg message) (*attribute, error) {
	walk := &cursor{data: msg.data}
	version := walk.u8()
	walk.skip(byteWide) // reserved
	nameSize, typeSize := int(walk.u16()), int(walk.u16())
	spaceSize := int(walk.u16())
	if walk.short {
		return nil, refuse(errCorrupt, msg.at, "IV.A.2.m",
			"attribute message of %d bytes is shorter than its "+
				"fixed part", len(msg.data))
	}
	if version != versionOneAttribute {
		return nil, refuse(errUnsupported, msg.at, "IV.A.2.m",
			"attribute version %d, only version 1 is read", version)
	}
	name := padded(walk, nameSize)
	kind := padded(walk, typeSize)
	space := padded(walk, spaceSize)
	if walk.short {
		return nil, refuse(errCorrupt, msg.at, "IV.A.2.m",
			"attribute name, datatype and dataspace do not fit in "+
				"%d bytes", len(msg.data))
	}
	rest := walk.take(len(msg.data) - walk.at)
	return buildAttribute(msg, name, kind, space, rest)
}

// padded takes so many bytes and then the padding that brings the
// position to a multiple of the attribute alignment.
func padded(walk *cursor, size int) []byte {
	got := walk.take(size)
	walk.align(attributeAlign)
	return got
}

// buildAttribute makes the attribute of the pieces of a message, and
// checks that the data is as long as the dataspace and datatype say.
func buildAttribute(msg message, name, kind, space,
	data []byte) (*attribute, error) {
	end := bytes.IndexByte(name, nameEnd)
	if end == notFound {
		return nil, refuse(errCorrupt, msg.at, "IV.A.2.m",
			"attribute name is not terminated")
	}
	typ, err := readDatatype(kind, msg.at)
	if err != nil {
		return nil, err
	}
	shape, err := readDataspace(space, msg.at)
	if err != nil {
		return nil, err
	}
	count, counted := elementCount(shape)
	want, fits := byteCount(count, typ.size)
	if !counted || !fits || want > uint64(len(data)) {
		return nil, refuse(errCorrupt, msg.at, "IV.A.2.m",
			"attribute %q: %d elements of %d bytes do not fit in "+
				"%d bytes", name[:end], count, typ.size,
			len(data))
	}
	return &attribute{named: named{name: string(name[:end]),
		shape: shape, kind: *typ}, raw: data[:want]}, nil
}

// attributes are the attributes in an object header, in file order.
func attributes(found *object) ([]*attribute, error) {
	var all []*attribute
	for _, msg := range found.messages {
		if msg.kind != messageAttribute {
			continue
		}
		made, err := readAttribute(msg)
		if err != nil {
			return nil, err
		}
		all = append(all, made)
	}
	return all, nil
}

// attributeNamed is the attribute of a name, or why there is none.
func attributeNamed(found *object, name string) (*attribute, error) {
	all, err := attributes(found)
	if err != nil {
		return nil, err
	}
	for _, each := range all {
		if each.name == name {
			return each, nil
		}
	}
	return nil, fmt.Errorf("%w: no attribute %q", errNotFound, name)
}
