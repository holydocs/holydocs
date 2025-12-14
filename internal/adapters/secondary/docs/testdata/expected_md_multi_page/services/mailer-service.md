# [←](../README.md) | Mailer Service
A service that handles email delivery through SendGrid. Receives email requests from other services and processes them for delivery. Supports various email types including transactional emails, notifications, and marketing campaigns.
- System: Notification System

- Owner: team-notifications

- Repository: [https://github.com/holydocs/mailer-service](https://github.com/holydocs/mailer-service)

- Tags: delivery, email, notifications, sendgrid


## Relationships

![Mailer Service Relationships](../diagrams/services/mailer-service-relationships.svg)
- **requests** SendGrid via SendGrid _(external)_ — A cloud-based email infrastructure platform that helps businesses send and manage
large volumes of transactional and marketing emails.

## Message Flow
![Mailer Service Service Interactions](../diagrams/services/mailer-service-service-services.svg)
