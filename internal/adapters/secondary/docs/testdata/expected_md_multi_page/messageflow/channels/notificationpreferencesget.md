# [←](../context.md) | notification.preferences.get

![notification.preferences.get](../../diagrams/messageflow/channel-notificationpreferencesget.svg)

## Messages
**request**: PreferencesRequestMessage
```json
{
  "user_id": "string[uuid]"
}
```
**reply**: PreferencesReplyMessage
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
  "updated_at": "string[date-time]"
}
```
