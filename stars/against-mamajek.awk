# Compare every row in stars/mainsequence.txt with the same spectral type
# in Mamajek’s published table. game run sources fetches the reference;
# game run extract invokes this check:
#
#   awk -f stars/against-mamajek.awk \
#     cited-sources/mamajek-dwarf-sequence-2022.txt stars/mainsequence.txt
#
# The embedded table claims unchanged source values. Mass, temperature,
# luminosity, and radius must therefore match character for character.
# Report missing or differing rows and fail if any are found.

# Read the first source table, finding columns by header name so a moved
# column does not silently change its meaning. Stop collecting at the
# second table header.
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

# Celestial columns: spectral type, mass, temperature, luminosity, radius.
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
