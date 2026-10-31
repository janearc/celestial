# against-iers.awk: holds every step of tai less utc in sky/leapseconds.json
# to the iers earth orientation centre's UTC-TAI.history, as game run
# sources fetches it. game run extract runs it:
#
#   awk -f sky/against-iers.awk \
#     sky/leapseconds.json cited-sources/iers-utc-tai-history.txt
#
# each step is the year and month it began and the whole seconds tai is
# ahead from then. it prints every step one has and the other does not,
# or has with other seconds, and fails if there is one.

# sky's steps: {"year": 1972, "month": 1, "ahead": 10}, two to a line.
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

# the history's steps from 1972, when utc took whole seconds: a line
# begins with its year, or with its month under the year above it, and
# ends with the seconds, as 11s. the lines before 1972 are rates, not
# steps, and are passed over.
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
