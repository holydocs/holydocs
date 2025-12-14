# [←](../context.md) | mailer.send

![mailer.send](../../diagrams/messageflow/channel-mailersend.svg)

## Messages
**receive**: EmailSendRequestMessage
```json
{
  "content": {
    "html": "string",
    "text": "string"
  },
  "email_id": "string[uuid]",
  "from": {
    "email": "string[email]",
    "name": "string"
  },
  "priority": "string[enum:low,normal,high]",
  "scheduled_at": "string[date-time]",
  "subject": "string",
  "template_data": "object",
  "template_id": "string",
  "to": [
    {
      "email": "string[email]",
      "name": "string"
    }
  ],
  "tracking": {
    "click_tracking": "boolean",
    "open_tracking": "boolean",
    "subscription_tracking": "boolean"
  }
}
```
