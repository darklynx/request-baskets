#!/bin/sh

if [ -z "$LISTEN" ]; then
    LISTEN="0.0.0.0"
fi

if [ -z "$DB" ]; then
    DB="bolt"
fi

if [ -z "$FILE" ]; then
    FILE="/var/lib/rbaskets/baskets.db"
fi

args="-l $LISTEN -db $DB -file $FILE"

if [ -n "$CONN" ]; then
    args="$args -conn $CONN"
fi

if [ -n "$PORT" ]; then
    args="$args -p $PORT"
fi

if [ -n "$PAGE" ]; then
    args="$args -page $PAGE"
fi

if [ -n "$SIZE" ]; then
    args="$args -size $SIZE"
fi

if [ -n "$MAXSIZE" ]; then
    args="$args -maxsize $MAXSIZE"
fi

if [ -n "$TOKEN" ]; then
    args="$args -token $TOKEN"
fi

if [ -n "$BASKET" ]; then
    args="$args -basket $BASKET"
fi

cmd="/bin/rbaskets $args"
echo "Executing: $cmd"
exec $cmd
