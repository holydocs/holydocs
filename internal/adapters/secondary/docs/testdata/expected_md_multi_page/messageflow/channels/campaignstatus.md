# [←](../context.md) | campaign.status

![campaign.status](../../diagrams/messageflow/channel-campaignstatus.svg)

## Messages
**send**: CampaignStatusUpdateMessage
```json
{
  "campaign_id": "string[uuid]",
  "error": {
    "code": "string",
    "message": "string"
  },
  "execution_id": "string[uuid]",
  "progress": {
    "failed": "integer",
    "sent": "integer",
    "success_rate": "number[float]",
    "total_targets": "integer"
  },
  "status": "string[enum:pending,running,completed,failed,paused,cancelled]",
  "updated_at": "string[date-time]"
}
```
