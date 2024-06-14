#!/usr/bin/env sh

if [[ $1 == "--help" || $1 == "-h" ]]; then
    echo ""
    echo "Welcome to 'mysqlsync'"
    echo "  - configuration file:"
    echo "     /etc/mysqlsync/config.yml"
    echo "  - option:"
    echo "     --replication"
    echo "     --destination"
    echo "  - prometheus export:"
    echo "    :9091/metrics"
    exit 0
fi

if [[ $1 == "--replication" ]]; then
    /app/replication
fi

if [[ $1 == "--destination" ]]; then
    /app/replication
fi