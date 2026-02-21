
TEMP_ACCESS_TOKEN=$(curl -s -X POST -H "Content-Type: application/json" \
                        -d '{"email": "admin@example.com", "password": "d1r3ctu5"}' \
                        $DIRECTUS_URL/auth/login \
                        | jq .data.access_token | cut -d '"' -f2)

USER_ID=$(curl -s -X GET -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TEMP_ACCESS_TOKEN" \
    $DIRECTUS_URL/users/me | jq .data.id | cut -d '"' -f2)

curl -s -X PATCH -H "Content-Type: application/json" \
    -H "Authorization: Bearer $TEMP_ACCESS_TOKEN" \
    -d "{\"token\": \"$ADMIN_ACCESS_TOKEN\"}" \
    $DIRECTUS_URL/users/$USER_ID > /dev/null

echo "Creating notifybot_chat_settings collection..."
curl -s -X POST -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ADMIN_ACCESS_TOKEN" \
    -d '{
        "collection": "notifybot_chat_settings",
        "fields": [
            {
                "field": "chat_id",
                "type": "string",
                "meta": {
                    "hidden": true,
                    "interface": "input",
                    "readonly": true
                },
                "schema": {
                    "is_primary_key": true
                }
            },
            {
                "field": "date_created",
                "type": "timestamp",
                "meta": {
                    "special": ["date-created"],
                    "interface": "datetime",
                    "readonly": true,
                    "hidden": true,
                    "width": "half",
                    "display": "datetime",
                    "display_options": {"relative": true}
                },
                "schema": {}
            }
        ],
        "schema": {},
        "meta": {"singleton": false}
    }' \
    $DIRECTUS_URL/collections | jq .

echo "Creating notifybot_currency_subscriptions collection..."
curl -s -X POST -H "Content-Type: application/json" \
    -H "Authorization: Bearer $ADMIN_ACCESS_TOKEN" \
    -d '{
        "collection": "notifybot_currency_subscriptions",
        "fields": [
            {
                "field": "id",
                "type": "uuid",
                "meta": {
                    "hidden": true,
                    "interface": "input",
                    "readonly": true,
                    "special": ["uuid"]
                },
                "schema": {
                    "is_primary_key": true
                }
            },
            {
                "field": "chat_id",
                "type": "string",
                "meta": {
                    "interface": "input",
                    "width": "half"
                },
                "schema": {
                    "is_nullable": false
                }
            },
            {
                "field": "currency",
                "type": "string",
                "meta": {
                    "interface": "input",
                    "width": "half"
                },
                "schema": {
                    "is_nullable": false
                }
            },
            {
                "field": "threshold_above",
                "type": "float",
                "meta": {
                    "interface": "input",
                    "width": "half",
                    "special": ["cast-decimal"]
                },
                "schema": {
                    "is_nullable": true
                }
            },
            {
                "field": "threshold_below",
                "type": "float",
                "meta": {
                    "interface": "input",
                    "width": "half",
                    "special": ["cast-decimal"]
                },
                "schema": {
                    "is_nullable": true
                }
            },
            {
                "field": "interval",
                "type": "float",
                "meta": {
                    "interface": "input",
                    "width": "half",
                    "special": ["cast-decimal"]
                },
                "schema": {
                    "is_nullable": true
                }
            },
            {
                "field": "last_notified_rate",
                "type": "float",
                "meta": {
                    "interface": "input",
                    "width": "half",
                    "special": ["cast-decimal"]
                },
                "schema": {
                    "default_value": 0,
                    "is_nullable": false
                }
            },
            {
                "field": "last_notification_time",
                "type": "timestamp",
                "meta": {
                    "interface": "datetime",
                    "width": "half"
                },
                "schema": {
                    "is_nullable": true
                }
            },
            {
                "field": "enabled",
                "type": "boolean",
                "meta": {
                    "interface": "boolean",
                    "width": "half",
                    "display": "boolean"
                },
                "schema": {
                    "default_value": true,
                    "is_nullable": false
                }
            },
            {
                "field": "date_created",
                "type": "timestamp",
                "meta": {
                    "special": ["date-created"],
                    "interface": "datetime",
                    "readonly": true,
                    "hidden": true,
                    "width": "half",
                    "display": "datetime",
                    "display_options": {"relative": true}
                },
                "schema": {}
            },
            {
                "field": "date_updated",
                "type": "timestamp",
                "meta": {
                    "special": ["date-updated"],
                    "interface": "datetime",
                    "readonly": true,
                    "hidden": true,
                    "width": "half",
                    "display": "datetime",
                    "display_options": {"relative": true}
                },
                "schema": {}
            }
        ],
        "schema": {},
        "meta": {"singleton": false}
    }' \
    $DIRECTUS_URL/collections | jq .

echo "Schema creation complete!"
