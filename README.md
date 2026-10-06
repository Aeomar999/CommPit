# mocksms

A local-first sandbox messaging provider for development. Apps send SMS, OTP and email to it instead of a real provider; nothing is delivered. Messages appear in a live web inbox, are readable via a test API, and run through a realistic lifecycle with provider-format webhooks.

**Current status:** v0.1.0 (M1) — in development

## Quick start

```bash
# Install (once binaries are published)
# brew install aeomar999/tap/mocksms
# scoop install mocksms

# Or run from source
go run ./cmd/mocksms serve
```

The server starts on `http://127.0.0.1:4010` with the web inbox at `/` and SMTP on `:1025`.

### With Docker

```bash
docker run -d \
  -p 4010:4010 \
  -p 1025:1025 \
  -v mocksms-data:/data \
  ghcr.io/aeomar999/commpit/mocksms:latest
```

### With Docker Compose

```yaml
# docker-compose.yml
version: '3.8'
services:
  mocksms:
    image: ghcr.io/aeomar999/commpit/mocksms:latest
    ports:
      - "4010:4010"
      - "1025:1025"
    volumes:
      - mocksms-data:/data
    environment:
      - MOCKSMS_HTTP_HOST=0.0.0.0
      - MOCKSMS_HTTP_PORT=4010
      - MOCKSMS_SMTP_HOST=0.0.0.0
      - MOCKSMS_SMTP_PORT=1025

volumes:
  mocksms-data:
```

```bash
docker-compose up -d
```

### From Source

```bash
# Prerequisites: Go 1.26+, Node.js 20+, pnpm
git clone https://github.com/Aeomar999/CommPit.git
cd CommPit

# Build and run
go run ./cmd/mocksms serve

# Or build binary
go build -o mocksms ./cmd/mocksms
./mocksms serve
```

## Features (Wave 1)

- **Native API** — Clean REST API for SMS, email, and verifications (`/api/v1/*`)
- **SMTP listener** — Receive email from any framework on `:1025`
- **Web inbox** — Live-updating UI with SMS threads, email viewer, OTP codes, filters
- **Test API** — `messages/wait`, `otp/latest`, `emails/latest` for reliable E2E tests
- **Lifecycle** — Messages progress through `queued` → `sent` → `delivered` (or `failed`)
- **Webhooks** — Status callbacks and inbound SMS in provider formats (M3)

## SMTP Setup Guides

mocksms accepts email via SMTP on port 1025 (default). No authentication required for local development.

### Laravel

```php
// config/mail.php
'default' => env('MAIL_MAILER', 'smtp'),

'mailers' => [
    'smtp' => [
        'transport' => 'smtp',
        'host' => env('MAIL_HOST', '127.0.0.1'),
        'port' => env('MAIL_PORT', 1025),
        'encryption' => env('MAIL_ENCRYPTION', null),
        'username' => env('MAIL_USERNAME', 'mocksms'),
        'password' => env('MAIL_PASSWORD', 'mocksms'),
    ],
],
```

```env
# .env
MAIL_MAILER=smtp
MAIL_HOST=127.0.0.1
MAIL_PORT=1025
MAIL_USERNAME=mocksms
MAIL_PASSWORD=mocksms
MAIL_ENCRYPTION=null
MAIL_FROM_ADDRESS="noreply@example.com"
MAIL_FROM_NAME="mocksms"
```

### Django

```python
# settings.py
EMAIL_BACKEND = 'django.core.mail.backends.smtp.EmailBackend'
EMAIL_HOST = '127.0.0.1'
EMAIL_PORT = 1025
EMAIL_USE_TLS = False
EMAIL_USE_SSL = False
EMAIL_HOST_USER = 'mocksms'
EMAIL_HOST_PASSWORD = 'mocksms'
DEFAULT_FROM_EMAIL = 'noreply@example.com'
```

### Rails

```ruby
# config/environments/development.rb
config.action_mailer.delivery_method = :smtp
config.action_mailer.smtp_settings = {
  address: '127.0.0.1',
  port: 1025,
  domain: 'localhost',
  user_name: 'mocksms',
  password: 'mocksms',
  authentication: :plain,
  enable_starttls_auto: false
}
```

### Nodemailer (Node.js)

```javascript
const nodemailer = require('nodemailer');

const transporter = nodemailer.createTransport({
  host: '127.0.0.1',
  port: 1025,
  secure: false, // true for 465, false for other ports
  auth: {
    user: 'mocksms',
    pass: 'mocksms'
  },
  tls: {
    rejectUnauthorized: false
  }
});

// Verify connection
transporter.verify((error, success) => {
  if (error) {
    console.log('SMTP connection failed:', error);
  } else {
    console.log('SMTP server is ready');
  }
});

// Send email
await transporter.sendMail({
  from: '"Test" <noreply@example.com>',
  to: 'user@example.com',
  subject: 'Test from mocksms',
  text: 'Hello from mocksms!',
  html: '<p>Hello from <strong>mocksms</strong>!</p>'
});
```

### Spring Boot

```yaml
# application.yml
spring:
  mail:
    host: 127.0.0.1
    port: 1025
    username: mocksms
    password: mocksms
    properties:
      mail:
        smtp:
          auth: true
          starttls:
            enable: false
      mail.smtp.ssl.enable: false
```

```java
// Or Java config
@Configuration
public class MailConfig {
    @Bean
    public JavaMailSender javaMailSender() {
        JavaMailSenderImpl mailSender = new JavaMailSenderImpl();
        mailSender.setHost("127.0.0.1");
        mailSender.setPort(1025);
        mailSender.setUsername("mocksms");
        mailSender.setPassword("mocksms");
        
        Properties props = mailSender.getJavaMailProperties();
        props.put("mail.smtp.auth", "true");
        props.put("mail.smtp.starttls.enable", "false");
        props.put("mail.smtp.ssl.enable", "false");
        
        return mailSender;
    }
}
```

## API Quick Reference

### Send SMS

```bash
curl -X POST http://127.0.0.1:4010/api/v1/sms \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "from": "+15551234567",
    "to": "+15559876543",
    "body": "Hello from mocksms!"
  }'
```

### Send Email

```bash
curl -X POST http://127.0.0.1:4010/api/v1/email \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "from": "sender@example.com",
    "to": ["recipient@example.com"],
    "subject": "Test Email",
    "text": "Hello from mocksms!",
    "html": "<p>Hello from <strong>mocksms</strong>!</p>"
  }'
```

### Start Verification (OTP)

```bash
curl -X POST http://127.0.0.1:4010/api/v1/verifications \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{
    "to": "+15559876543",
    "channel": "sms",
    "code_length": 6,
    "ttl_seconds": 600,
    "max_attempts": 5
  }'
```

### Check Verification

```bash
curl -X POST http://127.0.0.1:4010/api/v1/verifications/VRF_XXXXXX/check \
  -H "Content-Type: application/json" \
  -H "Authorization: Bearer YOUR_API_KEY" \
  -d '{"code": "123456"}'
```

### Wait for Message (Test API)

```bash
# Wait for SMS to +15559876543 (default 10s timeout)
curl "http://127.0.0.1:4010/api/v1/messages/wait?to=%2B15559876543&channel=sms"
```

### Web Inbox

Open `http://127.0.0.1:4010` in your browser to see the live inbox with:
- Message list with real-time updates via SSE
- SMS threads and email viewer
- OTP code extraction
- Message filtering and search
- Settings page

## Configuration

Create `mocksms.yaml` in current directory or `~/.config/mocksms/mocksms.yaml`:

```yaml
http:
  host: 127.0.0.1
  port: 4010
smtp:
  host: 127.0.0.1
  port: 1025
data_dir: ~/.local/share/mocksms
memory: false
lifecycle:
  step_delay: 300ms
otp:
  fixed_code: ""
validation:
  phone: valid
ui_auth: ""
```

### Environment Variables

All settings can be overridden via environment variables with `MOCKSMS_` prefix:

```bash
MOCKSMS_HTTP_PORT=4010
MOCKSMS_SMTP_PORT=1025
MOCKSMS_DATA_DIR=/var/lib/mocksms
MOCKSMS_MEMORY=false
MOCKSMS_LIFECYCLE_STEP_DELAY=300ms
MOCKSMS_VALIDATION_PHONE=valid
MOCKSMS_UI_AUTH=user:pass
```

## License

Apache-2.0 © 2026 Aeomar999

> **Note:** The Go module and GitHub repository are named `CommPit`. The product and binary are named `mocksms`. This is intentional — `CommPit` is the repository/organization name only.