#!/bin/sh

# De-linting is a time-consuming process. The aim of LINTY is to support an
# iterative process to clear out lint. It uses a configuration file which lists
# packages that currently contain lint, and ensures that:
#
#  - packages listed in the configuration are removed once they are free of lint
#  - packages not listed in the configuration continue to be free of lint
#
# If either of the above statements is FALSE, LINTY prints out a warning and
# exits. If both statements are TRUE, LINTY prints out a table of lint counts
# for the packages that are listed in its configuration.

if ! which golint >/dev/null; then
	echo "ERROR: golint not found!"
	echo "Please install by running 'go get golang.org/x/lint/golint'"
	exit 3
fi

# Configuration file
linty_config=".linty.conf"

# Temporary table file
tmp_file=$(mktemp /tmp/linty.XXXXXX)

for pkg in `go list ./...`; do
	# Check package for lint
	lint=$(golint -set_exit_status ${pkg} 2>/dev/null)
	has_lint=$?

	# Check if the package is expected to have lint
	if grep -Fxq $pkg ".linty.conf"; then
		if [ "$has_lint" -eq 1 ]; then
			# Still has lint...
			lint_count=$(echo "$lint" | wc -l)
			printf " %5s | %s\n" "$lint_count" "$pkg" >> "$tmp_file"
		else
			# Lint free!
			echo "ERROR: package $pkg contains NO lint, but is listed in the LINTY config."
			echo "Please remove it from '$linty_config'!"
			rm $tmp_file
			exit 1
		fi
	else
		if [ "$has_lint" -eq 1 ]; then
			# New lint...
			echo "$lint"
			echo ""
			echo "ERROR: package $pkg contains NEW lint. Please address the issues listed above!"
			rm $tmp_file
			exit 2
		fi
	fi
done

# Sort results by count
sort -nr $tmp_file -o $tmp_file

# Print results table
echo "================================  L I N T Y   W A L L   O F   S H A M E  ================================"
echo ""
echo " Count | Name of Linty Package"
echo "-------+-------------------------------------------------------------------------------------------------"
cat $tmp_file
echo ""
echo "Help LINTY fight the good fight, golint today!"

# Remove temporary file
rm $tmp_file
