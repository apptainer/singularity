#!/bin/sh -
set -e


# this is only useful for C projects
if [ $lang_c -eq 0 ]; then
	return
fi


confh=$builddir/config.h
confc=$builddir/config.c

touch $confh $confc

config_h_add_char ()
{
	if [ "$1" = "" -o "$2" = "" ]; then
		echo "error: config_h_add_char: not called with <var_name> <var_value>"
		return
	fi
	name=$1
	value=$2

	echo "char config_h_get_$name(void);" >> $confh
	cat >> $confc << EOF
char
config_h_get_$name(void)
{
	return $value;
}
EOF
}

config_h_add_short ()
{
	if [ "$1" = "" -o "$2" = "" ]; then
		echo "error: config_h_add_short: not called with <var_name> <var_value>"
		return
	fi
	name=$1
	value=$2

	echo "short config_h_get_$name(void);" >> $confh
	cat >> $confc << EOF
short
config_h_get_$name(void)
{
	return $value;
}
EOF
}

config_h_add_int ()
{
	if [ "$1" = "" -o "$2" = "" ]; then
		echo "error: config_h_add_int: not called with <var_name> <var_value>"
		return
	fi
	name=$1
	value=$2

	echo "int config_h_get_$name(void);" >> $confh
	cat >> $confc << EOF
int
config_h_get_$name(void)
{
	return $value;
}
EOF
}

config_h_add_long ()
{
	if [ "$1" = "" -o "$2" = "" ]; then
		echo "error: config_h_add_long: not called with <var_name> <var_value>"
		return
	fi
	name=$1
	value=$2

	echo "long config_h_get_$name(void);" >> $confh
	cat >> $confc << EOF
long
config_h_get_$name(void)
{
	return $value;
}
EOF
}

config_h_add_longlong ()
{
	if [ "$1" = "" -o "$2" = "" ]; then
		echo "error: config_h_add_longlong: not called with <var_name> <var_value>"
		return
	fi
	name=$1
	value=$2

	echo "long long config_h_get_$name(void);" >> $confh
	cat >> $confc << EOF
long long
config_h_get_$name(void)
{
	return $value;
}
EOF
}

config_h_add_string ()
{
	if [ "$1" = "" -o "$2" = "" ]; then
		echo "error: config_h_add_string: not called with <var_name> <var_value>"
		return
	fi
	name=$1
	value=$2

	echo "char *config_h_get_$name(void);" >> $confh
	cat >> $confc << EOF
char *
config_h_get_$name(void)
{
	return "$value";
}
EOF
}

config_add_def ()
{
        if [ "$1" = "" -o "$2" = "" ]; then
                return
        fi
        echo "#define $*" >> $confh
}

config_add_header ()
{
	echo "#ifndef __CONFIG_H_" >> $confh
	echo "#define __CONFIG_H_" >> $confh
	echo >> $confh
}

config_add_footer ()
{
	echo >> $confh
	echo "#endif /* __CONFIG_H_ */" >> $confh
}

