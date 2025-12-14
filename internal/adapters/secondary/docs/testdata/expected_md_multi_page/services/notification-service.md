# [←](../README.md) | Notification Service
A service that handles user notifications, preferences, and interactions. Supports real-time notifications, user preferences management.
- System: Notification System

- Owner: team-notifications

- Repository: [https://github.com/holydocs/notification-service](https://github.com/holydocs/notification-service)

- Tags: notifications, preferences, real-time


## Relationships

![Notification Service Relationships](../diagrams/services/notification-service-relationships.svg)
- **requests** Firebase Cloud Messaging via FCM _(external)_ — A service from Google that enables developers to send notifications and
data messages to Android, iOS, and web apps

## Inter-Service Connections
- sends to Analytics Service via notification.analytics
- receives from Campaign Service via notification.user.{user_id}.push
- receives from User Service via user.info.request (reply)
- receives from User Service via notification.preferences.update
- sends to User Service via user.info.request
## Message Flow
![Notification Service Service Interactions](../diagrams/services/notification-service-service-services.svg)
- publishes to Analytics Service (pub)
- receives from Campaign Service (pub)
- receives from User Service (pub)
- requests to User Service (req)
### Related Channels
- [notification.analytics](../messageflow/channels/notificationanalytics.md)
- [notification.preferences.update](../messageflow/channels/notificationpreferencesupdate.md)
- [notification.user.{user_id}.push](../messageflow/channels/notificationuseruser-idpush.md)
- [user.info.request](../messageflow/channels/userinforequest.md)
