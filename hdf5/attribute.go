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

// attribute stores a parsed description and little-endian value bytes from the
// attribute message.
type attribute struct {
	named
	raw []byte
}

// readAttribute parses a version 1 Attribute message (IV.A.2.m). Truncated
// messages and unsupported versions return an error.
func readAttribute(msg message) (*attribute, error) {
	walk := &cursor{data: msg.data}
	version := walk.u8()
	walk.skip(byteWide) // reserved
	nameSize, typeSize := int(walk.u16()), int(walk.u16())
	spaceSize := int(walk.u16())
	if walk.short {
		return nil, fmt.Errorf("hdf5: file is damaged: attribute "+
			"message of %d bytes is shorter than its fixed part, "+
			"at address %#x (spec "+
			"IV.A.2.m)", len(msg.data), msg.at)
	}
	if version != versionOneAttribute {
		return nil, fmt.Errorf("hdf5: part of the format not read: "+
			"attribute version %d, only version 1 is read, at "+
			"address %#x (spec IV.A.2.m)", version, msg.at)
	}
	name := padded(walk, nameSize)
	kind := padded(walk, typeSize)
	space := padded(walk, spaceSize)
	if walk.short {
		return nil, fmt.Errorf("hdf5: file is damaged: attribute "+
			"name, datatype and dataspace do not fit in %d "+
			"bytes, at address %#x (spec "+
			"IV.A.2.m)", len(msg.data), msg.at)
	}
	rest := walk.take(len(msg.data) - walk.at)
	return buildAttribute(msg, name, kind, space, rest)
}

// padded reads a field and advances to the attribute-alignment boundary.
func padded(walk *cursor, size int) []byte {
	got := walk.take(size)
	walk.align(attributeAlign)
	return got
}

// buildAttribute assembles parsed fields and checks that value bytes match the
// dataspace and datatype sizes.
func buildAttribute(msg message, name, kind, space,
	data []byte) (*attribute, error) {
	end := bytes.IndexByte(name, nameEnd)
	if end == notFound {
		return nil, fmt.Errorf("hdf5: file is damaged: attribute "+
			"name is not terminated, at address %#x (spec "+
			"IV.A.2.m)", msg.at)
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
		return nil, fmt.Errorf("hdf5: file is damaged: attribute %q: "+
			"%d elements of %d bytes do not fit in %d bytes, at "+
			"address %#x (spec IV.A.2.m)", name[:end], count,
			typ.size, len(data), msg.at)
	}
	return &attribute{named: named{name: string(name[:end]),
		shape: shape, kind: *typ}, raw: data[:want]}, nil
}

// attributes returns header attributes in file order.
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

// attributeNamed looks up an attribute or reports why it is unavailable.
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
	return nil, fmt.Errorf("hdf5: not found: no attribute %q", name)
}
