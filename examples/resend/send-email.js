// Redirect resend to mocksms. Run with:
//   MOCKSMS_URL=http://127.0.0.1:4010 node send-email.js
const { Resend } = require('resend');

const base = process.env.MOCKSMS_URL || 'http://127.0.0.1:4010';
const resend = new Resend(process.env.RESEND_API_KEY || 're_mock-key', { baseUrl: base + '/resend' });

(async () => {
  const { data, error } = await resend.emails.send({
    from: 'sender@example.com',
    to: 'alice@example.com',
    subject: 'Hello from mocksms',
    text: 'Your verification code is 482916',
  });
  if (error) { console.error(error); process.exit(1); }
  console.log('id:', data.id);
  console.log('E2E-OK');
})();