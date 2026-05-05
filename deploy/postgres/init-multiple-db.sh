#!/bin/bash
set -e

psql -v ON_ERROR_STOP=1 --username "$POSTGRES_USER" --dbname postgres <<-EOSQL
    CREATE DATABASE $BOT_DB;
    CREATE DATABASE $SCRAPPER_DB;
EOSQL