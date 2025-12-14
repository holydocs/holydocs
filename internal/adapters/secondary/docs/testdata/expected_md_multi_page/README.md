# HolyDOCs Test Documentation

## Table of Contents

- [Overview](#overview)
- [Services](#services)
  - [Analytics System](systems/analytics-system.md)
    - [Analytics Service](services/analytics-service.md)
    - [Reports Service](services/reports-service.md)
  - [Notification System](systems/notification-system.md)
    - [Mailer Service](services/mailer-service.md)
    - [Notification Service](services/notification-service.md)
  - [Standalone Services](systems/standalone-services.md)
    - [Campaign Service](services/campaign-service.md)
    - [User Service](services/user-service.md)
- [Message Flow](messageflow/context.md)
  - [Context](messageflow/context.md#context)
  - [Channels](messageflow/context.md#channels)
    - [analytics.alert](messageflow/channels/analyticsalert.md)
    - [analytics.insights](messageflow/channels/analyticsinsights.md)
    - [analytics.report.request](messageflow/channels/analyticsreportrequest.md)
    - [campaign.analytics](messageflow/channels/campaignanalytics.md)
    - [campaign.create](messageflow/channels/campaigncreate.md)
    - [campaign.execute](messageflow/channels/campaignexecute.md)
    - [campaign.status](messageflow/channels/campaignstatus.md)
    - [mailer.batch](messageflow/channels/mailerbatch.md)
    - [mailer.send](messageflow/channels/mailersend.md)
    - [notification.analytics](messageflow/channels/notificationanalytics.md)
    - [notification.preferences.get](messageflow/channels/notificationpreferencesget.md)
    - [notification.preferences.update](messageflow/channels/notificationpreferencesupdate.md)
    - [notification.user.{user_id}.push](messageflow/channels/notificationuseruser-idpush.md)
    - [reports.delivery](messageflow/channels/reportsdelivery.md)
    - [reports.scheduled](messageflow/channels/reportsscheduled.md)
    - [user.analytics](messageflow/channels/useranalytics.md)
    - [user.info.request](messageflow/channels/userinforequest.md)
    - [user.info.update](messageflow/channels/userinfoupdate.md)

## Overview

![Overview](diagrams/overview.svg)

### Design Principles
- **Event-driven architecture**: Services communicate through async message queues
- **Microservices with clear boundaries**: Each service has a single responsibility
- **Async communication**: All inter-service communication is asynchronous
- **Data-driven insights**: Analytics service provides real-time insights and reporting

### Technology Stack
- **Message Queues**: AsyncAPI for event-driven communication
- **Databases**: ClickHouse for analytics, PostgreSQL for transactional data
- **External Services**: SendGrid for email, Firebase for push notifications
- **Monitoring**: Built-in analytics and reporting capabilities

