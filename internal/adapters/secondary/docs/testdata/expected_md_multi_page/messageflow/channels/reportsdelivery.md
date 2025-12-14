# [←](../context.md) | reports.delivery

![reports.delivery](../../diagrams/messageflow/channel-reportsdelivery.svg)

## Messages
**send**: ReportDeliveryMessage
```json
{
  "attachment_url": "string[uri]",
  "delivered_at": "string[date-time]",
  "delivery_id": "string[uuid]",
  "delivery_method": "string[enum:email,webhook,s3,ftp]",
  "error_message": "string",
  "recipient": "string[email]",
  "report_id": "string[uuid]",
  "status": "string[enum:pending,sent,delivered,failed]"
}
```
