#!/bin/bash
# Add a template/policy/index

set -e

ES_URL=${ES_URL:-"http://localhost:9200"}
ES_USER=${ES_USER:-"elastic"}
ES_PASSWORD=${ES_PASSWORD:-"upy3RV72S6p8h73oJn2TQ1z6"}
TEMPLATE_NAME=${TEMPLATE_NAME:-"otlp_spans"}
INDEX_PATTERN="${TEMPLATE_NAME}-*"
INDEX_ALIAS="${TEMPLATE_NAME}-alias"

SHARD_COUNT=2
REPLICA_COUNT=0
REFRESH_INTERVAL="5s"
TRANSLOG_DURABILITY="async"

function init() {
    echo "creating insight es template/policy/index..."
    while [[ "$(curl --insecure -u "$ES_USER:$ES_PASSWORD" -s -o /dev/null -w '%{http_code}\n' $ES_URL)" != "200" ]]; do sleep 1; done
    # k8s logs es ilm policy
    curl --insecure -u "$ES_USER:$ES_PASSWORD" -XPUT "$ES_URL/_index_template/${TEMPLATE_NAME}-template" -H 'Content-Type: application/json' -d'{"index_patterns": ['\""$INDEX_PATTERN"\"'],"template": {"settings":{ "index": {"lifecycle": {"name": '\""${TEMPLATE_NAME}-policy"\"',"rollover_alias": '\""${INDEX_ALIAS}"\"'}, "refresh_interval": '\""$REFRESH_INTERVAL"\"',"translog":{"durability": '\""$TRANSLOG_DURABILITY"\"'},"number_of_shards": '$SHARD_COUNT',"number_of_replicas": '$REPLICA_COUNT'}}}}'
    curl --insecure -u "$ES_USER:$ES_PASSWORD" -XPUT "$ES_URL/_ilm/policy/${TEMPLATE_NAME}-policy" -H 'Content-Type: application/json' -d'{"policy":{"phases":{"hot":{"min_age":"0ms","actions":{"forcemerge":{"max_num_segments":1},"rollover":{"max_primary_shard_size":"10gb", "max_age" : "30d", "max_size" : "20gb"}}},"delete":{"min_age":"3d","actions":{"delete":{"delete_searchable_snapshot":true}}}}}}'
    curl --insecure -u "$ES_USER:$ES_PASSWORD" -XPUT "$ES_URL/${TEMPLATE_NAME}-000001" -H 'Content-Type: application/json' -d'{"aliases": {'\""${INDEX_ALIAS}"\"':{"is_write_index": true }}}'
}

./es_clean.sh

init
