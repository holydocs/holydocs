# [←](../README.md) | Analytics Service
A centralized analytics service that receives and processes analytics events from all other services. Provides insights, reporting, and analytics data aggregation for user behavior, notification performance, campaign effectiveness, and system-wide metrics.
- System: Analytics System

- Owner: team-data-science

- Repository: [https://github.com/holydocs/analytics-service](https://github.com/holydocs/analytics-service)

- Tags: analytics, data-science


## Relationships

![Analytics Service Relationships](../diagrams/services/analytics-service-relationships.svg)
- **replies** Data Analyst via http-server (http) — A data analyst who is responsible for analyzing data and providing insights.

- **uses** clickhouse via ClickHouse — Uses ClickHouse database
## Inter-Service Connections
- receives from Campaign Service via campaign.analytics
- receives from Notification Service via notification.analytics
- receives from Reports Service via analytics.report.request
- replies to Reports Service via analytics.report.request (reply)
- receives from User Service via user.analytics
## Message Flow
![Analytics Service Service Interactions](../diagrams/services/analytics-service-service-services.svg)
- receives from Campaign Service (pub)
- receives from Notification Service (pub)
- handles requests from Reports Service (req)
- receives from User Service (pub)
### Related Channels
- [analytics.report.request](../messageflow/channels/analyticsreportrequest.md)
- [campaign.analytics](../messageflow/channels/campaignanalytics.md)
- [notification.analytics](../messageflow/channels/notificationanalytics.md)
- [user.analytics](../messageflow/channels/useranalytics.md)
