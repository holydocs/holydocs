# [←](../context.md) | analytics.alert

![analytics.alert](../../diagrams/messageflow/channel-analyticsalert.svg)

## Messages
**send**: AnalyticsAlertMessage
```json
{
  "actions": [
    "string"
  ],
  "affected_services": [
    "string[enum:user_service,notification_service,campaign_service]"
  ],
  "alert_id": "string[uuid]",
  "alert_type": "string[enum:anomaly_detected,threshold_exceeded,trend_change,system_issue]",
  "created_at": "string[date-time]",
  "current_value": "number",
  "description": "string",
  "metadata": {
    "environment": "string[enum:development,staging,production]",
    "platform": "string[enum:ios,android,web]",
    "source": "string[enum:mobile,web,api]",
    "version": "string"
  },
  "metric": "string",
  "severity": "string[enum:low,medium,high,critical]",
  "threshold": "number",
  "time_window": "string",
  "title": "string"
}
```
