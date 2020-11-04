msg="$*"
curl -X POST -H 'Content-Type: application/json' \
 -d '{"chat_id": "29462028", "text": "'"$msg"'", "disable_notification": true, "parse_mode":"markdown"}' \
 https://api.telegram.org/bot"$DEPLOYMENT_BOT_TOKEN"/sendMessage