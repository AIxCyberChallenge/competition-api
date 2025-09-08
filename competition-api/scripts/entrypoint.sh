#! /bin/sh

program="./$1"
shift

exec "$program" "${@}"
