# [←](../context.md) | notification.preferences.update

![notification.preferences.update](../../diagrams/messageflow/channel-notificationpreferencesupdate.svg)

## Messages
**receive**: PreferencesUpdateMessage
```json
{
  "preferences": {
    "categories": {
      "marketing": "boolean",
      "security": "boolean",
      "updates": "boolean"
    },
    "email_enabled": "boolean",
    "push_enabled": "boolean",
    "quiet_hours": {
      "enabled": "boolean",
      "end": "string[time]",
      "start": "string[time]"
    },
    "sms_enabled": "boolean"
  },
  "updated_at": "string[date-time]",
  "user_id": "string[uuid]"
}
```
