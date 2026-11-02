# Compare TAI−UTC steps in sky/leapseconds.json with IERS’s UTC-TAI.history.
# game run sources fetches the reference; game run extract runs:
#
#   awk -f sky/against-iers.awk \
#     sky/leapseconds.json cited-sources/iers-utc-tai-history.txt
#
# Each entry records its starting year and month and the whole seconds
# TAI leads UTC. Report missing steps or differing offsets in either
# input and fail if any are found.

# Sky entries have year, month, and ahead fields, two records per line.
FNR == NR {
	line = $0
	while (match(line, /"year": [0-9]+, "month": [0-9]+, "ahead": [0-9]+/)) {
		step = substr(line, RSTART, RLENGTH)
		line = substr(line, RSTART + RLENGTH)
		gsub(/[^0-9,]/, "", step)
		split(step, part, ",")
		sky[part[1] "-" part[2]] = part[3]
	}
	next
}

# Read steps from 1972, when UTC began using whole SI seconds. Rows start
# with a year or a month continuing the previous year, and end with an
# offset such as 11s. Earlier entries describe rates, not these steps.
$1 ~ /^[0-9][0-9][0-9][0-9]$/ {
	year = $1
	month = $2
}
$1 !~ /^[0-9][0-9][0-9][0-9]$/ {
	month = $1
}
year >= 1972 && $NF ~ /^[0-9]+s$/ {
	number = (substr(month, 1, 3) == "Jan") ? 1 : 7
	ahead = $NF
	sub(/s$/, "", ahead)
	iers[year "-" number] = ahead
}

END {
	for (step in iers) {
		steps++
		if (!(step in sky)) {
			print "sky: " step " is " iers[step] " s in iers, and not in sky"
			wrong++
		} else if (sky[step] != iers[step]) {
			print "sky: " step " is " sky[step] " s; iers has " iers[step]
			wrong++
		}
	}
	for (step in sky) {
		if (!(step in iers)) {
			print "sky: " step " is in sky, and not in iers"
			wrong++
		}
	}
	if (wrong) {
		print "sky: " wrong " steps differ from iers's"
		exit 1
	}
	print "sky: all " steps " steps are iers's"
}
