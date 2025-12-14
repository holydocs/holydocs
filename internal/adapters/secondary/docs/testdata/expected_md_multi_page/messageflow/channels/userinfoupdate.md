# [←](../context.md) | user.info.update

![user.info.update](../../diagrams/messageflow/channel-userinfoupdate.svg)

## Messages
**send**: UserInfoUpdateMessage
```json
{
  "changes": "object",
  "metadata": {
    "environment": "string[enum:development,staging,production]",
    "platform": "string[enum:ios,android,web]",
    "source": "string[enum:mobile,web,api]",
    "version": "string"
  },
  "updated_at": "string[date-time]",
  "user_id": "string[uuid]"
}
```
