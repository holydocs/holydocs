# [←](../context.md) | reports.scheduled

![reports.scheduled](../../diagrams/messageflow/channel-reportsscheduled.svg)

## Messages
**send**: ScheduledReportMessage
```json
{
  "next_run": "string[date-time]",
  "recipients": [
    "string[email]"
  ],
  "report_type": "string[enum:user_activity,notification_performance,campaign_effectiveness,system_health,custom]",
  "schedule": {
    "frequency": "string[enum:daily,weekly,monthly,quarterly,yearly]",
    "time": "string[time]",
    "timezone": "string"
  },
  "schedule_id": "string[uuid]"
}
```
