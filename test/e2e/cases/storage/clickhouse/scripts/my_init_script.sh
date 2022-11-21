#!/bin/bash
set -e

clickhouse client -u default --password='changeme' -n <<-EOSQL
  CREATE DATABASE openinsight;
EOSQL