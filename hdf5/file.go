package hdf5

import (
	"errors"
	"fmt"
	"io"
	"strings"
)

// File reads on demand from its io.ReaderAt, retaining the parsed superblock
// rather than copying the whole file.
type File struct {
	from  *source
	super *superblock
}

// noSize is no bytes at all. Defined: a file shorter than nothing is none.
const noSize = 0

// NewFile validates a reader and byte length as a supported HDF5 file. Negative
// sizes, missing signatures, and unsupported format features return an error
// and no File. Format errors identify the feature and its location.
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

// resolve finds an object header by slash-separated path. Empty path and /
// identify the root group.
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
			return nil, fmt.Errorf("hdf5: not found: %q is no "+
				"group, so holds no %q", pathSeparator+walked,
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

// noAddress is what findLink gives beside a refusal. Defined: none.
const noAddress = 0

// findLink finds a named group member’s object-header address.
func findLink(links []link, name string) (uint64, error) {
	for _, member := range links {
		if member.name == name {
			return member.header, nil
		}
	}
	return noAddress, errors.New("hdf5: not found: no member of that name")
}

// errNotGroup is what groupLinks gives for an object with no members to
// list. Members checks for it, and then says the path is not found.
var errNotGroup = errors.New("hdf5: not that kind of value: not a group")

// groupLinks lists an old-style group through its symbol table message. An
// object without that message is not a supported group.
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
