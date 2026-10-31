# against-mamajek.awk: holds every row of stars/mainsequence.txt to the
# row of the same type in mamajek's table, as game run sources fetches it.
# game run extract runs it:
#
#   awk -f stars/against-mamajek.awk \
#     cited-sources/mamajek-dwarf-sequence-2022.txt stars/mainsequence.txt
#
# mainsequence.txt says its rows are mamajek's, unchanged, so each of its
# mass, temperature, luminosity and radius must be the source's text,
# character for character. it prints every row that differs, or is
# missing from the source, and fails if there is one.

# the source's first table: its columns found by name from its header,
# so a column moved in a later version is still found, and its rows kept
# until the second table's header.
FNR == NR && /^#SpT/ {
	headers++
	if (headers == 1) {
		for (i = 1; i <= NF; i++) {
			if (!($i in at)) {
				at[$i] = i
			}
		}
	}
	next
}
FNR == NR {
	if (headers == 1 && $0 !~ /^#/ && NF > 1) {
		teff[$1] = $(at["Teff"])
		logl[$1] = $(at["logL"])
		radius[$1] = $(at["R_Rsun"])
		mass[$1] = $(at["Msun"])
	}
	next
}

# celestial's rows: type, mass, temperature, luminosity, radius.
/^#/ || NF == 0 { next }
{
	rows++
	if (!($1 in teff)) {
		print "stars: " $1 " is not in mamajek's table"
		wrong++
		next
	}
	if ($2 != mass[$1] || $3 != teff[$1] || $4 != logl[$1] ||
	    $5 != radius[$1]) {
		print "stars: " $1 " is " $2 " " $3 " " $4 " " $5 \
		    "; mamajek has " mass[$1] " " teff[$1] " " logl[$1] " " \
		    radius[$1]
		wrong++
	}
}
END {
	if (wrong) {
		print "stars: " wrong " of " rows " rows differ from mamajek's"
		exit 1
	}
	print "stars: all " rows " rows are mamajek's, character for character"
}
