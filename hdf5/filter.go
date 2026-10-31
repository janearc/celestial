package hdf5

// versionOneFilters is the version of the Filter Pipeline message that is
// read. Defined by IV.A.2.l.
const versionOneFilters = 1

// The filter identification numbers of IV.A.2.l's table of filter ids,
// each defined by it.
const (
	filterDeflate    = 1 // defined by IV.A.2.l
	filterShuffle    = 2 // defined by IV.A.2.l
	filterFletcher32 = 3 // defined by IV.A.2.l
	filterSzip       = 4 // defined by IV.A.2.l
	filterNbit       = 5 // defined by IV.A.2.l
	filterScaleOff   = 6 // defined by IV.A.2.l
)

// The fixed parts of a version 1 Filter Pipeline message and of each
// filter description in it, each defined by IV.A.2.l: after the version
// and the number of filters, 6 reserved bytes; after each filter's id
// and name length, its flags and the number of client data values.
//
// The flags are 2 bytes, of which bit 0 is "optional", which changes
// nothing here. Each value is four bytes, and an odd count of them is
// padded by one more.
const (
	filtersReserved = 6 // defined by IV.A.2.l
	flagsSize       = 2 // defined by IV.A.2.l
	clientValueSize = 4 // defined by IV.A.2.l
	pairs           = 2 // defined: an even count pads nothing
	oddOne          = 1 // defined: what an odd count leaves over
	noID            = 0 // defined: no filter has id 0
)

// filters is a dataset's filter pipeline, in the order the message lists
// it, which is the order a writer applies them.
type filters struct {
	ids []int
}

// readFilters parses a Filter Pipeline message. IV.A.2.l, version 1. It
// refuses a message cut short, a version not read, and a filter not
// undone here, and then gives none.
func readFilters(msg message) (*filters, error) {
	walk := &cursor{data: msg.data}
	version, count := walk.u8(), int(walk.u8())
	walk.skip(filtersReserved)
	if walk.short {
		return nil, refuse(errCorrupt, msg.at, "IV.A.2.l",
			"filter pipeline message is cut short")
	}
	if version != versionOneFilters {
		return nil, refuse(errUnsupported, msg.at, "IV.A.2.l",
			"filter pipeline version %d, only version 1 is read",
			version)
	}
	found := &filters{}
	for range count {
		id, err := readFilter(walk, msg.at)
		if err != nil {
			return nil, err
		}
		found.ids = append(found.ids, id)
	}
	return found, nil
}

// readFilter reads one filter description and gives its id, if it is
// one this reader undoes: deflate and Fletcher32.
func readFilter(walk *cursor, at uint64) (int, error) {
	id, nameLength := int(walk.u16()), int(walk.u16())
	walk.skip(flagsSize)
	values := int(walk.u16())
	walk.skip(nameLength)
	walk.skip(values * clientValueSize)
	if values%pairs == oddOne {
		walk.skip(clientValueSize)
	}
	if walk.short {
		return noID, refuse(errCorrupt, at, "IV.A.2.l",
			"filter description is cut short")
	}
	switch id {
	case filterDeflate, filterFletcher32:
		return id, nil
	}
	return noID, refuse(errUnsupported, at, "IV.A.2.l",
		"filter %d (%s), only deflate and Fletcher32 are read", id,
		filterName(id))
}

// filterNames is the name IV.A.2.l gives each filter id refused.
var filterNames = map[int]string{filterShuffle: "shuffle",
	filterSzip: "szip", filterNbit: "nbit", filterScaleOff: "scale-offset"}

// filterName is the name IV.A.2.l gives a filter id.
func filterName(id int) string {
	if name, held := filterNames[id]; held {
		return name
	}
	return "not in the specification's table"
}
