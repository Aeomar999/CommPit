import { Mail, Copy, Download, Eye, Clock, AlertCircle, CheckCircle, XCircle, MessageSquare } from 'lucide-react'
import { cn } from '../lib/utils'
import { Button } from '../components/ui/Button'

interface Message {
  id: string
  from: string
  to: string
  subject: string
  body: string
  status: 'queued' | 'sent' | 'delivered' | 'undelivered' | 'failed' | 'received'
  channel: 'sms' | 'email'
  createdAt: string
}

const mockMessages: Message[] = [
  {
    id: 'msg_1',
    from: '+15551234567',
    to: '+15559876543',
    subject: '',
    body: 'Hello, this is a test message!',
    status: 'delivered',
    channel: 'sms',
    createdAt: '2024-01-15T10:30:00Z',
  },
  {
    id: 'msg_2',
    from: 'sender@example.com',
    to: 'recipient@example.com',
    subject: 'Welcome to mocksms',
    body: 'Welcome to mocksms! This is a test email.',
    status: 'sent',
    channel: 'email',
    createdAt: '2024-01-15T10:25:00Z',
  },
  {
    id: 'msg_3',
    from: '+15551112222',
    to: '+15553334444',
    subject: '',
    body: 'Your verification code is 123456',
    status: 'failed',
    channel: 'sms',
    createdAt: '2024-01-15T10:20:00Z',
  },
]

const statusColors = {
  queued: 'bg-yellow-100 text-yellow-800',
  sent: 'bg-blue-100 text-blue-800',
  delivered: 'bg-green-100 text-green-800',
  undelivered: 'bg-orange-100 text-orange-800',
  failed: 'bg-red-100 text-red-800',
  received: 'bg-purple-100 text-purple-800',
}

const statusIcons = {
  queued: Clock,
  sent: Mail,
  delivered: CheckCircle,
  undelivered: AlertCircle,
  failed: XCircle,
  received: MessageSquare,
}

export function HomePage() {
  return (
    <div className="space-y-6">
      <div className="flex items-center justify-between">
        <div>
          <h1 className="text-3xl font-bold tracking-tight">Inbox</h1>
          <p className="text-muted-foreground">All your messages in one place</p>
        </div>
<div className="flex gap-2">
          <Button variant="outline">Filters</Button>
          <Button>Compose</Button>
        </div>
      </div>

      <div className="rounded-md border bg-card">
        <div className="overflow-x-auto">
          <table className="w-full caption-bottom text-sm">
            <thead className="[&_tr]:border-b">
              <tr className="border-b transition-colors hover:bg-muted/50 data-[state=selected]:bg-muted">
                <th className="h-12 px-4 text-left align-middle font-medium text-muted-foreground">From</th>
                <th className="h-12 px-4 text-left align-middle font-medium text-muted-foreground">To</th>
                <th className="h-12 px-4 text-left align-middle font-medium text-muted-foreground">Subject / Preview</th>
                <th className="h-12 px-4 text-left align-middle font-medium text-muted-foreground">Channel</th>
                <th className="h-12 px-4 text-left align-middle font-medium text-muted-foreground">Status</th>
                <th className="h-12 px-4 text-left align-middle font-medium text-muted-foreground">Time</th>
                <th className="h-12 px-4 text-left align-middle font-medium text-muted-foreground">Actions</th>
              </tr>
            </thead>
            <tbody>
              {mockMessages.map((msg) => {
                const Icon = statusIcons[msg.status]
                return (
                  <tr key={msg.id} className="border-b transition-colors hover:bg-muted/50">
                    <td className="p-4 font-medium">{msg.from}</td>
                    <td className="p-4 text-muted-foreground">{msg.to}</td>
                    <td className="p-4 max-w-xs truncate">
                      {msg.subject || msg.body.slice(0, 50) + '...'}
                    </td>
                    <td className="p-4">
                      <span className={cn('inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium', msg.channel === 'sms' ? 'bg-blue-100 text-blue-800' : 'bg-green-100 text-green-800')}>
                        {msg.channel === 'sms' ? 'SMS' : 'Email'}
                      </span>
                    </td>
                    <td className="p-4">
                      <span className={cn('inline-flex items-center gap-1 px-2 py-1 rounded-full text-xs font-medium', statusColors[msg.status])}>
                        <Icon className="w-3 h-3" />
                        {msg.status}
                      </span>
                    </td>
                    <td className="p-4 text-muted-foreground">
                      {new Date(msg.createdAt).toLocaleString()}
                    </td>
                    <td className="p-4">
                      <div className="flex items-center gap-2">
                        <button className="p-2 hover:bg-accent rounded-md" title="View"><Eye className="w-4 h-4" /></button>
                        <button className="p-2 hover:bg-accent rounded-md" title="Copy"><Copy className="w-4 h-4" /></button>
                        <button className="p-2 hover:bg-accent rounded-md" title="Download"><Download className="w-4 h-4" /></button>
                      </div>
                    </td>
                  </tr>
                )
              }
              )}
            </tbody>
          </table>
        </div>
      </div>

      <div className="flex items-center justify-between py-4">
        <p className="text-sm text-muted-foreground">Showing 1-3 of 3 messages</p>
        <div className="flex gap-2">
          <Button variant="outline" disabled>Previous</Button>
          <Button variant="outline" disabled>Next</Button>
        </div>
      </div>
    </div>
  )
}