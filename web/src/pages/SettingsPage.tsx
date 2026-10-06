import { Save, Key, Globe, Trash2, Shield } from 'lucide-react'

export function SettingsPage() {
  return (
    <div className="space-y-6 max-w-3xl">
      <div>
        <h1 className="text-3xl font-bold tracking-tight">Settings</h1>
        <p className="text-muted-foreground">Manage your mocksms configuration</p>
      </div>

      <div className="space-y-6">
        <section className="space-y-4">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <Globe className="w-5 h-5" />
            General
          </h2>
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <label htmlFor="project-name" className="text-sm font-medium">Project Name</label>
              <input
                id="project-name"
                type="text"
                defaultValue="mocksms"
                className="input"
                placeholder="Enter project name"
              />
            </div>
            <div className="space-y-2">
              <label htmlFor="timezone" className="text-sm font-medium">Timezone</label>
              <select id="timezone" className="input">
                <option value="UTC">UTC</option>
                <option value="America/New_York">Eastern Time</option>
                <option value="America/Los_Angeles">Pacific Time</option>
              </select>
            </div>
          </div>
        </section>

        <section className="space-y-4">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <Key className="w-5 h-5" />
            Authentication
          </h2>
          <div className="space-y-4">
            <div className="space-y-2">
              <label htmlFor="ui-auth" className="text-sm font-medium">UI Basic Auth (user:pass)</label>
              <input
                id="ui-auth"
                type="text"
                placeholder="admin:password"
                className="input"
              />
              <p className="text-sm text-muted-foreground">Leave empty to disable. Format: username:password</p>
            </div>
          </div>
        </section>

        <section className="space-y-4">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <Shield className="w-5 h-5" />
            Security
          </h2>
          <div className="space-y-4">
            <div className="space-y-2">
              <label className="text-sm font-medium">Allowed Hosts</label>
              <textarea
                placeholder="localhost&#10;127.0.0.1&#10;example.com"
                className="input min-h-[80px]"
                defaultValue="127.0.0.1\nlocalhost"
              />
              <p className="text-sm text-muted-foreground">One host per line. Empty = allow all.</p>
            </div>
            <div className="flex items-center space-x-2">
              <input type="checkbox" id="require-x-mocksms" defaultChecked />
              <label htmlFor="require-x-mocksms" className="text-sm">
                Require X-Mocksms header for API requests
              </label>
            </div>
          </div>
        </section>

        <section className="space-y-4">
          <h2 className="text-lg font-semibold flex items-center gap-2">
            <Trash2 className="w-5 h-5 text-destructive" />
            Data Retention
          </h2>
          <div className="grid gap-4 md:grid-cols-2">
            <div className="space-y-2">
              <label className="text-sm font-medium">Message TTL</label>
              <select className="input" defaultValue="30d">
                <option value="7d">7 days</option>
                <option value="30d">30 days</option>
                <option value="90d">90 days</option>
              </select>
            </div>
            <div className="space-y-2">
              <label className="text-sm font-medium">Prune Interval</label>
              <select className="input" defaultValue="1h">
                <option value="1h">1 hour</option>
                <option value="6h">6 hours</option>
                <option value="24h">24 hours</option>
              </select>
            </div>
          </div>
        </section>

        <div className="flex justify-end pt-4 border-t">
          <button className="btn btn-primary flex items-center gap-2">
            <Save className="w-4 h-4" />
            Save Changes
          </button>
        </div>
      </div>
    </div>
  )
}