#!/bin/bash
# clean insight template/policy/index

set -e

ES_URL=${ES_URL:-"http://localhost:9200"}
ES_USER=${ES_USER:-"elastic"}
ES_PASSWORD=${ES_PASSWORD:-"upy3RV72S6p8h73oJn2TQ1z6"}
TEMPLATE_NAME=${TEMPLATE_NAME:-"otlp_spans"}
INDEX_PATTERN="${TEMPLATE_NAME}-*"

function clean_openinsight_index() {
    echo "cleaning insight es template/policy/index..."
    while [[ "$(curl --insecure -u "$ES_USER:$ES_PASSWORD" -s -o /dev/null -w '%{http_code}\n' $ES_URL)" != "200" ]]; do sleep 1; done
    # clean index
    curl --insecure -u "$ES_USER:$ES_PASSWORD" -XDELETE "$ES_URL/${INDEX_PATTERN}"
    # clean templte
    curl --insecure -u "$ES_USER:$ES_PASSWORD" -XDELETE "$ES_URL/_index_template/${TEMPLATE_NAME}*"
    # clean alias
    curl --insecure -u "$ES_USER:$ES_PASSWORD" -XDELETE "$ES_URL/otlp_spans-*"
    # clean ilm policy
    curl --insecure -u "$ES_USER:$ES_PASSWORD" -XDELETE "$ES_URL/_ilm/policy/${TEMPLATE_NAME}-policy"
}

clean_openinsight_index
