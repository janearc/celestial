package hdf5

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// File is an open HDF5 file. It reads from the io.ReaderAt it was given,
// when asked, and holds nothing of the file but its superblock.
type File struct {
	from  *source
	super *superblock
}

// noSize is no bytes at all. Defined: a file shorter than nothing is none.
const noSize = 0

// NewFile opens the HDF5 file a reader holds, so many bytes long. It
// refuses a file that is not HDF5 with ErrNotHDF5, one that uses a part of
// the format this reader does not read with ErrUnsupported, saying which
// part and where, and a size below nothing, and then gives no file.
func NewFile(reader io.ReaderAt, size int64) (*File, error) {
	if size < noSize {
		return nil, fmt.Errorf("hdf5: a file cannot be %d bytes long",
			size)
	}
	from := &source{reader: reader, size: uint64(size)}
	super, err := readSuperblock(from)
	if err != nil {
		return nil, err
	}
	return &File{from: from, super: super}, nil
}

// pathSeparator joins the names of a path. Defined by HDF5's paths.
const pathSeparator = "/"

// resolve is the object header at a path of names joined by slashes. The
// empty path and "/" are the root group.
func (file *File) resolve(path string) (*object, error) {
	at := file.super.rootHeader
	walked := ""
	trimmed := strings.Trim(path, pathSeparator)
	for _, name := range strings.Split(trimmed, pathSeparator) {
		if name == "" {
			continue
		}
		parent, err := readObject(file.from, at)
		if err != nil {
			return nil, err
		}
		links, err := file.groupLinks(parent)
		if errors.Is(err, errNotGroup) {
			return nil, fmt.Errorf("%w: %q is no group, so holds "+
				"no %q", errNotFound, pathSeparator+walked,
				name)
		}
		if err != nil {
			return nil, fmt.Errorf("%q: %w", walked, err)
		}
		at, err = findLink(links, name)
		if err != nil {
			return nil, fmt.Errorf("%q in %q: %w", name,
				pathSeparator+walked, err)
		}
		walked += pathSeparator + name
	}
	return readObject(file.from, at)
}

// errNoMember is the cause when a group has no member of a name.
var errNoMember = fmt.Errorf("%w: no member of that name", errNotFound)

// noAddress is what findLink gives beside a refusal. Defined: none.
const noAddress = 0

// findLink is the object header address of the member of a name.
func findLink(links []link, name string) (uint64, error) {
	for _, member := range links {
		if member.name == name {
			return member.header, nil
		}
	}
	return noAddress, errNoMember
}

// errNotGroup is the cause when members are asked of an object that has
// none: the wrong kind of object.
var errNotGroup = fmt.Errorf("%w: not a group", errWrongType)

// groupLinks is the members of a group object. An object with no symbol
// table message is not a group this reader can list.
func (file *File) groupLinks(group *object) ([]link, error) {
	for _, msg := range group.messages {
		if msg.kind != messageSymbolTable {
			continue
		}
		table, err := parseSymbolTable(msg)
		if err != nil {
			return nil, err
		}
		return file.from.members(*table, file.super.leafK)
	}
	return nil, errNotGroup
}
