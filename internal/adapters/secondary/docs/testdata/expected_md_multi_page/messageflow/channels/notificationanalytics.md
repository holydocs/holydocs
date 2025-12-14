# [←](../context.md) | notification.analytics

![notification.analytics](../../diagrams/messageflow/channel-notificationanalytics.svg)

## Messages
**receive**: NotificationAnalyticsEventMessage
```json
{
  "event_id": "string[uuid]",
  "event_type": "string[enum:notification_sent,notification_opened,notification_clicked]",
  "metadata": {
    "environment": "string[enum:development,staging,production]",
    "platform": "string[enum:ios,android,web]",
    "source": "string[enum:mobile,web,api]",
    "version": "string"
  },
  "notification_id": "string[uuid]",
  "timestamp": "string[date-time]",
  "user_id": "string[uuid]"
}
```
