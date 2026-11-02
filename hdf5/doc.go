// Package hdf5 reads the HDF5 structures used by the open science centre’s
// strain files, following the HDF5 File Format Specification, version 3.0.
//
// Supported files have a version 0 superblock with eight-byte offsets and
// lengths, version 1 object headers and continuations, and old-style groups
// using version 1 B-trees, symbol table nodes, and local name heaps.
//
// Datasets and attributes may contain little-endian integers, IEEE binary32
// and binary64 floats, and fixed- or variable-length strings. Storage may
// be contiguous or chunked; chunks may use deflate and Fletcher-32, whose
// checksum is verified.
//
// Other features return errors. Errors explain unsupported structures,
// damaged data, or a missing HDF5 signature in words, with the address and
// specification section where applicable. Failed reads return no result.
package hdf5
