# [←](../context.md) | analytics.report.request

![analytics.report.request](../../diagrams/messageflow/channel-analyticsreportrequest.svg)

## Messages
**request**: AnalyticsReportRequestMessage
```json
{
  "created_at": "string[date-time]",
  "filters": {
    "campaign_ids": [
      "string[uuid]"
    ],
    "event_types": [
      "string"
    ],
    "user_ids": [
      "string[uuid]"
    ],
    "user_segments": [
      "string[enum:all_users,new_users,active_users,inactive_users,premium_users,free_users]"
    ]
  },
  "format": "string[enum:json,csv,pdf]",
  "metrics": [
    "string[enum:event_count,user_count,conversion_rate,engagement_rate,response_time,error_rate]"
  ],
  "report_id": "string[uuid]",
  "report_type": "string[enum:user_activity,notification_performance,campaign_effectiveness,system_health,custom]",
  "time_range": {
    "end": "string[date-time]",
    "granularity": "string[enum:minute,hour,day,week,month]",
    "start": "string[date-time]"
  }
}
```
**reply**: AnalyticsReportReplyMessage
```json
{
  "data": "object",
  "error": {
    "code": "string",
    "message": "string"
  },
  "generated_at": "string[date-time]",
  "insights": [
    {
      "confidence": "number[float]",
      "data_points": [
        "object"
      ],
      "description": "string",
      "impact": "string[enum:low,medium,high]",
      "title": "string",
      "type": "string[enum:trend,anomaly,correlation,recommendation]"
    }
  ],
  "report_id": "string[uuid]",
  "report_type": "string[enum:user_activity,notification_performance,campaign_effectiveness,system_health,custom]",
  "summary": {
    "event_types": "object",
    "top_metrics": {
      "conversion_rate": "number[float]",
      "engagement_rate": "number[float]",
      "error_rate": "number[float]",
      "response_time_avg": "number[float]"
    },
    "total_events": "integer",
    "unique_users": "integer"
  },
  "time_range": {
    "end": "string[date-time]",
    "granularity": "string[enum:minute,hour,day,week,month]",
    "start": "string[date-time]"
  }
}
```
