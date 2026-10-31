// Package hdf5 reads the parts of the HDF5 file format that the open
// science centre's strain files use, as the HDF5 File Format
// Specification, version 3.0, describes them, and refuses the rest.
//
// It reads a version 0 superblock with eight-byte offsets and lengths,
// version 1 object headers and their continuations, and old-style
// groups: a version 1 B-tree of symbol table nodes over a local heap of
// names.
//
// It reads datasets and attributes of little-endian integers, binary32
// and binary64 floats, and fixed and variable-length strings.
//
// A dataset is contiguous or chunked, and a chunk may be deflated and
// may carry a Fletcher-32 checksum, which is checked.
//
// Anything else is refused with ErrUnsupported, which names the part
// and its section of the specification. Bytes that contradict the
// specification or themselves are refused with ErrCorrupt, and bytes
// that are not HDF5 at all with ErrNotHDF5. A refusal gives nothing.
package hdf5
