# [←](../context.md) | mailer.batch

![mailer.batch](../../diagrams/messageflow/channel-mailerbatch.svg)

## Messages
**receive**: BatchEmailRequestMessage
```json
{
  "batch_id": "string[uuid]",
  "batch_settings": {
    "delay_between_batches": "integer",
    "max_concurrent": "integer"
  },
  "emails": [
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
      ]
    }
  ]
}
```
