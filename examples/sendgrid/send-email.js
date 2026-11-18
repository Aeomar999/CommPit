// Redirect @sendgrid/mail to mocksms. Run with:
//   MOCKSMS_URL=http://127.0.0.1:4010 node send-email.js
const sgMail = require('@sendgrid/mail');

const base = process.env.MOCKSMS_URL || 'http://127.0.0.1:4010';
sgMail.setApiKey(process.env.SENDGRID_API_KEY || 'SG.mock-key');

// Override the client base URL so requests land on mocksms.
const client = sgMail.Client ? new sgMail.Client({ baseUrl: base + '/sendgrid' }) : null;
if (client) sgMail.setClient(client);

(async () => {
  const msg = {
    to: 'alice@example.com',
    from: 'sender@example.com',
    subject: 'Hello from mocksms',
    text: 'Your verification code is 482916',
  };
  const [res] = await sgMail.send(msg);
  console.log('status:', res.statusCode, 'x-message-id:', res.headers['x-message-id']);
  console.log('E2E-OK');
})();