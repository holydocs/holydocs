# [←](../README.md) | Reports Service
A service that generates and manages analytics reports by requesting data from the analytics service. Provides report scheduling, customization, and delivery capabilities for business intelligence and data-driven decision making.
- System: Analytics System

- Owner: team-data-science

- Repository: [https://github.com/holydocs/reports-service](https://github.com/holydocs/reports-service)

- Tags: analytics, business-intelligence, reporting


## Relationships

![Reports Service Relationships](../diagrams/services/reports-service-relationships.svg)
_No relationships documented._
## Inter-Service Connections
- receives from Analytics Service via analytics.report.request (reply)
- sends to Analytics Service via analytics.report.request
## Message Flow
![Reports Service Service Interactions](../diagrams/services/reports-service-service-services.svg)
- requests to Analytics Service (req)
### Related Channels
- [analytics.report.request](../messageflow/channels/analyticsreportrequest.md)
