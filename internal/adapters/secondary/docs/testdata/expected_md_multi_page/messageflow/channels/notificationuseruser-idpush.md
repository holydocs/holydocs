# [←](../context.md) | notification.user.{user_id}.push

![notification.user.{user_id}.push](../../diagrams/messageflow/channel-notificationuseruser-idpush.svg)

## Messages
**receive**: PushNotificationMessage
```json
{
  "body": "string",
  "created_at": "string[date-time]",
  "data": "object",
  "notification_id": "string[uuid]",
  "priority": "string[enum:low,normal,high]",
  "title": "string",
  "user_id": "string[uuid]"
}
```
