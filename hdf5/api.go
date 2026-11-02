package hdf5

import (
	"slices"
)

// Values holds a dataset or attribute’s shape and stored bytes, decoded into a
// requested value type on demand.
type Values struct {
	shape []uint64
	read  values
	from  *source
}

// Read loads the dataset at a slash-separated path such as strain/Strain.
// Missing paths, objects that are not datasets, and unsupported storage return
// an error with no values.
func (file *File) Read(path string) (*Values, error) {
	found, err := file.resolve(path)
	if err != nil {
		return nil, err
	}
	return file.valuesOf(found)
}

// valuesOf reads dataset elements from their object header, rejecting
// non-datasets and unsupported representations.
func (file *File) valuesOf(found *object) (*Values, error) {
	set, err := datasetOf(found)
	if err != nil {
		return nil, err
	}
	raw, err := set.raw(file.from)
	if err != nil {
		return nil, err
	}
	return &Values{shape: set.shape, read: values{typ: set.kind, raw: raw},
		from: file.from}, nil
}

// Attribute reads a named attribute from a group or dataset, such as Xspacing
// on strain/Strain. Missing paths or names return an error.
func (file *File) Attribute(path, name string) (*Values, error) {
	found, err := file.resolve(path)
	if err != nil {
		return nil, err
	}
	held, err := attributeNamed(found, name)
	if err != nil {
		return nil, err
	}
	return &Values{shape: held.shape, read: values{typ: held.kind,
		raw: held.raw}, from: file.from}, nil
}

// Members lists a group’s members in file order. Empty path and / identify the
// root. Missing paths and non-group objects return an error.
func (file *File) Members(path string) ([]string, error) {
	found, err := file.resolve(path)
	if err != nil {
		return nil, err
	}
	links, err := file.groupLinks(found)
	if err != nil {
		return nil, err
	}
	names := make([]string, noBytes, len(links))
	for _, each := range links {
		names = append(names, each.name)
	}
	slices.Sort(names)
	return names, nil
}

// Shape returns dimension sizes. A scalar has no dimensions.
func (held *Values) Shape() []uint64 { return slices.Clone(held.shape) }

// Floats decodes floating-point elements in file order and rejects other
// element types.
func (held *Values) Floats() ([]float64, error) {
	return held.read.float64s()
}

// Integers decodes integer elements in file order and rejects other element
// types.
func (held *Values) Integers() ([]int64, error) {
	return held.read.int64s()
}

// Texts decodes fixed- or variable-length strings in file order and rejects
// other element types.
func (held *Values) Texts() ([]string, error) {
	return held.read.strings(held.from)
}
