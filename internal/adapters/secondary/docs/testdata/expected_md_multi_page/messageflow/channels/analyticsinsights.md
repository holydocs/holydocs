# [←](../context.md) | analytics.insights

![analytics.insights](../../diagrams/messageflow/channel-analyticsinsights.svg)

## Messages
**send**: AnalyticsInsightMessage
```json
{
  "category": "string[enum:user_behavior,notification_performance,campaign_effectiveness,system_health]",
  "confidence": "number[float]",
  "created_at": "string[date-time]",
  "data_points": [
    "object"
  ],
  "description": "string",
  "insight_id": "string[uuid]",
  "insight_type": "string[enum:trend,anomaly,recommendation,alert]",
  "metadata": {
    "environment": "string[enum:development,staging,production]",
    "platform": "string[enum:ios,android,web]",
    "source": "string[enum:mobile,web,api]",
    "version": "string"
  },
  "recommendations": [
    "string"
  ],
  "severity": "string[enum:low,medium,high,critical]",
  "title": "string"
}
```
