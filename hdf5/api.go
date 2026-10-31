package hdf5

import (
	"slices"
)

// Values is the elements of a dataset or an attribute as read: their
// shape, and the bytes the file holds, decoded when asked for in a kind.
type Values struct {
	shape []uint64
	read  values
	from  *source
}

// Read is the elements of the dataset at a path of names joined by
// slashes, such as strain/Strain. It refuses a path the file does not
// have, an object there that is no dataset, and a dataset this reader
// cannot read, and then gives none.
func (file *File) Read(path string) (*Values, error) {
	found, err := file.resolve(path)
	if err != nil {
		return nil, err
	}
	return file.valuesOf(found)
}

// valuesOf is the elements of the dataset an object header describes. It
// refuses an object that is no dataset, and a dataset this reader cannot
// read, and then gives none.
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

// Attribute is the elements of an attribute of the group or dataset at a
// path, by its name, such as Xspacing on strain/Strain. It refuses a path
// or a name the file does not have, and then gives none.
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

// Members is the names of the members of the group at a path, in order.
// The path "" and "/" are the root group. It refuses a path the file does
// not have and an object there that is no group, and then gives none.
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

// Shape is the size of each of the values' dimensions; a scalar has none.
func (held *Values) Shape() []uint64 { return slices.Clone(held.shape) }

// Floats is the values as floats, in the file's order. It refuses values
// that are not floating-point, and then gives none.
func (held *Values) Floats() ([]float64, error) {
	return held.read.float64s()
}

// Integers is the values as integers, in the file's order. It refuses
// values that are not fixed-point, and then gives none.
func (held *Values) Integers() ([]int64, error) {
	return held.read.int64s()
}

// Texts is the values as text, fixed or variable in length, in the file's
// order. It refuses values that are not strings, and then gives none.
func (held *Values) Texts() ([]string, error) {
	return held.read.strings(held.from)
}
