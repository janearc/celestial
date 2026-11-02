package hdf5

import (
	"bytes"
	"fmt"
)

// signature is the eight bytes every HDF5 file starts with, as the
// specification's section II.A, Format Signature and Superblock, gives
// them: 0x89, "HDF", carriage return, line feed, 0x1a, line feed.
var signature = []byte("\x89HDF\r\n\x1a\n")

// The version 0 superblock read, each defined by II.A. Its length with
// eight-byte offsets and lengths is the fields of II.A up to and
// including the root group symbol table entry of III.C, which is 40
// bytes.
//
// II.A lets a file choose the size of an offset and of a length, and the
// open science centre's files choose eight. The version is 0, and so is
// each of the four version fields after it; the 4 bytes of consistency
// flags are unused in version 0; and the base address is 0.
const (
	superblock0Size   = 96 // defined by II.A
	addressSize       = 8  // defined by the files read
	superblockVersion = 0  // defined by II.A
	versionField      = 0  // defined by II.A: each version field is 0
	consistencyFlags  = 4  // defined by II.A
	noBase            = 0  // defined by II.A: the base address read
	noK               = 0  // defined by II.A: a K of nothing is damaged
	fileStart         = 0  // defined by II.A: a file with no user block
	backOne           = 1  // defined: the field just read
	backTwo           = 2  // defined: the two fields just read
	backFour          = 4  // defined: the two K fields just read
)

// superblock is what the reader keeps of the superblock: where the root
// group's object header is.
type superblock struct {
	// rootHeader is the address of the root group's object header, from
	// the root group symbol table entry (III.C).
	rootHeader uint64
	// leafK is the group leaf node K: a symbol table node holds at most
	// twice this many entries (III.B).
	leafK uint16
}

// readSuperblock validates the version 0 superblock at the start of the
// file (II.A). A missing signature returns an error identifying the input
// as non-HDF5.
func readSuperblock(from *source) (*superblock, error) {
	signed := uint64(len(signature))
	if from.size < signed {
		return nil, fmt.Errorf("hdf5: not an HDF5 file: %d bytes, "+
			"too few for the format signature, at address %#x "+
			"(spec II.A)", from.size, fileStart)
	}
	start, err := from.read(fileStart, signed, "II.A")
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(start, signature) {
		return nil, fmt.Errorf("hdf5: not an HDF5 file: the format "+
			"signature is %x, at address %#x (spec "+
			"II.A)", start, fileStart)
	}
	raw, err := from.read(fileStart, superblock0Size, "II.A")
	if err != nil {
		return nil, err
	}
	walk := &cursor{data: raw, at: len(signature)}
	version := walk.u8()
	if version != superblockVersion {
		return nil, fmt.Errorf("hdf5: part of the format not read: "+
			"superblock version %d, only version 0 is read, at "+
			"address %#x (spec II.A)", version, signed)
	}
	return readSuperblockFields(from, walk)
}

// readSuperblockFields checks the fields after the version byte and
// keeps the root header address.
func readSuperblockFields(from *source, walk *cursor) (*superblock,
	error) {
	// free-space version, root entry version, reserved, shared header
	// message version: each must be zero.
	for _, name := range []string{"free-space storage version",
		"root symbol table entry version", "reserved byte",
		"shared header message format version"} {
		if got := walk.u8(); got != versionField {
			field := uint64(walk.at - backOne)
			return nil, fmt.Errorf("hdf5: part of the format not "+
				"read: %s is %d, only 0 is read, at address "+
				"%#x (spec II.A)", name, got, field)
		}
	}
	offsets, lengths := walk.u8(), walk.u8()
	if offsets != addressSize || lengths != addressSize {
		field := uint64(walk.at - backTwo)
		return nil, fmt.Errorf("hdf5: part of the format not read: "+
			"size of offsets %d and of lengths %d, only %d is "+
			"read, at address %#x (spec "+
			"II.A)", offsets, lengths, addressSize, field)
	}
	walk.skip(byteWide) // reserved, zero
	leaf, internal := walk.u16(), walk.u16()
	if leaf == noK || internal == noK {
		field := uint64(walk.at - backFour)
		return nil, fmt.Errorf("hdf5: file is damaged: group leaf "+
			"node K %d or internal node K %d is zero, at address "+
			"%#x (spec II.A)", leaf, internal, field)
	}
	walk.skip(consistencyFlags)
	found, err := readSuperblockAddresses(from, walk)
	if err != nil {
		return nil, err
	}
	found.leafK = leaf
	return found, nil
}

// readSuperblockAddresses checks the four addresses and the root group
// symbol table entry, which end the version 0 superblock.
func readSuperblockAddresses(from *source, walk *cursor) (*superblock,
	error) {
	base := walk.u64()
	if base != noBase {
		field := uint64(walk.at - addressSize)
		return nil, fmt.Errorf("hdf5: part of the format not read: "+
			"base address %#x, only 0 is read, at address %#x "+
			"(spec II.A)", base, field)
	}
	walk.skip(addressSize) // free-space index, unused in version 0
	end := walk.u64()
	if end > from.size {
		field := uint64(walk.at - addressSize)
		return nil, fmt.Errorf("hdf5: file is damaged: end of file "+
			"address %d is past the file's %d bytes: truncated, "+
			"at address %#x (spec "+
			"II.A)", end, from.size, field)
	}
	driver := walk.u64()
	if driver != undefinedAddress {
		field := uint64(walk.at - addressSize)
		return nil, fmt.Errorf("hdf5: part of the format not read: a "+
			"driver information block at %#x, at address %#x "+
			"(spec II.A)", driver, field)
	}
	// Root group symbol table entry, III.C: link name offset, object
	// header address, cache type, 4 reserved bytes, 16 scratch-pad bytes.
	walk.skip(addressSize)
	return &superblock{rootHeader: walk.u64()}, nil
}
