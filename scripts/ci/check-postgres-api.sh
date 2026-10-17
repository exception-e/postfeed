#!/usr/bin/env bash
set -euo pipefail

: "${GRAPHQL_URL:?GRAPHQL_URL is required}"
: "${TEST_USER_ID:?TEST_USER_ID is required}"

graphql() {
  local payload="$1"
  local response

  response=$(curl \
    --silent \
    --show-error \
    --fail-with-body \
    --connect-timeout 2 \
    --max-time 15 \
    -H 'Content-Type: application/json' \
    -H "user-id: ${TEST_USER_ID}" \
    --data-binary "$payload" \
    "$GRAPHQL_URL") || return 1

  if ! jq -e '
    (type == "object")
    and (((.errors // []) | length) == 0)
    and (.data != null)
  ' <<< "$response" > /dev/null; then
    echo "GraphQL returned an invalid or unsuccessful response:" >&2
    echo "$response" >&2
    return 1
  fi

  printf '%s\n' "$response"
}

wait_for_graphql() {
  local deadline=$((SECONDS + 90))
  local payload='{"query":"query { __typename }"}'

  while (( SECONDS < deadline )); do
    if graphql "$payload" > /dev/null 2>&1; then
      echo "GraphQL is available"
      return 0
    fi

    sleep 2
  done

  echo "GraphQL did not become available" >&2
  return 1
}

assert_post_exists() {
  local response

  response=$(graphql "$read_payload") || return 1

  if ! jq -e \
    --arg id "$post_id" \
    --arg body "$post_body" \
    --arg user_id "$TEST_USER_ID" '
      any(
        .data.posts.nodes[];
        .id == $id
        and .body == $body
        and .userId == $user_id
        and .commentsEnabled == true
      )
    ' <<< "$response" > /dev/null; then
    echo "Expected post was not found:" >&2
    echo "$response" >&2
    return 1
  fi
}

wait_for_graphql

post_body="CI PostgreSQL check $(cat /proc/sys/kernel/random/uuid)"

create_query='
mutation CreatePost($input: CreatePostInput!) {
  createPost(input: $input) {
    id
    userId
    body
    commentsEnabled
  }
}
'

create_payload=$(jq -n \
  --arg query "$create_query" \
  --arg body "$post_body" '
  {
    query: $query,
    variables: {
      input: {
        body: $body,
        commentsEnabled: true
      }
    }
  }
')

echo "Creating a post"
created=$(graphql "$create_payload")

jq -e \
  --arg body "$post_body" \
  --arg user_id "$TEST_USER_ID" '
    .data.createPost
    | ((.id | type) == "string")
      and ((.id | length) > 0)
      and (.body == $body)
      and (.userId == $user_id)
      and (.commentsEnabled == true)
  ' <<< "$created" > /dev/null

post_id=$(jq -r '.data.createPost.id' <<< "$created")
echo "Created post: $post_id"

read_query='
query {
  posts(first: 50) {
    nodes {
      id
      userId
      body
      commentsEnabled
    }
  }
}
'

read_payload=$(jq -n \
  --arg query "$read_query" \
  '{query: $query}')

echo "Checking the post through the API"
assert_post_exists

echo "Restarting only the application"
docker compose --profile psql restart app

wait_for_graphql

echo "Checking persistence after application restart"
assert_post_exists

echo "PostgreSQL API check passed"