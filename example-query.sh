#!/bin/bash

curl https://api.anthropic.com/v1/messages \
     --show-headers \
     --header "x-api-key: $ANTHROPIC_API_KEY" \
     --header "anthropic-version: 2023-06-01" \
     --header "content-type: application/json" \
     --data \
'{
    "model": "claude-sonnet-4-0",
    "max_tokens": 2048,
    "system": "You are a helpful assistant that provides concise and accurate answers to user queries.  Your responses should short: no longer than 2 or 3 sentences.",
    "messages": [
        {"role": "user", "content": "Why is the sky blue?"}
    ]
}'

exit 0

# Example response JSON:
# 
# {
#     "id": "msg_01FLtpFj1qRsnPKs5UygTinR",
#     "type": "message",
#     "role": "assistant",
#     "model": "claude-sonnet-4-20250514",
#     "content": [
#         {   // Array element 0.  Only present when thinking is enabled.
#             "type": "thinking",
#             "thinking": "<THINKING TEXT HERE>...",
#             "signature": "WaUjzkypQ2mUEVM36O2TxuC06KN8xyfbJwyem2dw3UjavL...."
#         },
#         {   // Array element 1.
#             "type": "text",
#             "text": "<AI RESPONSE HERE>..."
#         }
#     ],
#     "stop_reason": "end_turn",
#     "stop_sequence": null,
#     "usage": {
#         "input_tokens": 23,
#         "cache_creation_input_tokens": 0,
#         "cache_read_input_tokens": 0,
#         "output_tokens": 127,
#         "service_tier": "standard"
#     }
# }
