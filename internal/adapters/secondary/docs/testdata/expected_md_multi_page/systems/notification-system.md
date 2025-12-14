# [←](../README.md) | Notification System
![Notification System](../diagrams/system-notification-system.svg)

#### Key Features
- **Multi-channel support**: Email, push notifications, SMS
- **User preferences**: Respects user notification preferences and quiet hours
- **Batch processing**: Efficient handling of large notification volumes
- **Real-time delivery**: Push notifications for immediate user engagement
- **Analytics integration**: Full tracking of notification performance and user engagement

## Services
### [Mailer Service](../services/mailer-service.md)
A service that handles email delivery through SendGrid. Receives email requests from other services and processes them for delivery. Supports various email types including transactional emails, notifications, and marketing campaigns.
- System: Notification System

- Owner: team-notifications

- Repository: [https://github.com/holydocs/mailer-service](https://github.com/holydocs/mailer-service)

- Tags: delivery, email, notifications, sendgrid

### [Notification Service](../services/notification-service.md)
A service that handles user notifications, preferences, and interactions. Supports real-time notifications, user preferences management.
- System: Notification System

- Owner: team-notifications

- Repository: [https://github.com/holydocs/notification-service](https://github.com/holydocs/notification-service)

- Tags: notifications, preferences, real-time

