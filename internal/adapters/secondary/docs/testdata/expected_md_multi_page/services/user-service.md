# [←](../README.md) | User Service
A service that manages user information, profiles, and authentication. Handles user data requests, profile updates, and user lifecycle events.

## Relationships

![User Service Relationships](../diagrams/services/user-service-relationships.svg)
- **uses** elasticsearch via Elasticsearch — Uses Elasticsearch database
- **uses** postgres via PostgreSQL — Uses PostgreSQL database
## Inter-Service Connections
- sends to Analytics Service via user.analytics
- receives from Campaign Service via user.info.request
- replies to Campaign Service via user.info.request (reply)
- receives from Notification Service via user.info.request
- replies to Notification Service via user.info.request (reply)
- sends to Notification Service via notification.preferences.update
## Message Flow
![User Service Service Interactions](../diagrams/services/user-service-service-services.svg)
- publishes to Analytics Service (pub)
- handles requests from Campaign Service (req)
- handles requests from Notification Service (req)
- publishes to Notification Service (pub)
### Related Channels
- [notification.preferences.update](../messageflow/channels/notificationpreferencesupdate.md)
- [user.analytics](../messageflow/channels/useranalytics.md)
- [user.info.request](../messageflow/channels/userinforequest.md)
