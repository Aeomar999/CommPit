import { Home, RotateCcw } from "lucide-react";
import { Link } from "react-router-dom";
import { Button } from "../components/ui/Button";

export function NotFoundPage() {
  return (
    <div className="flex min-h-[60vh] flex-col items-center justify-center text-center">
      <h1 className="text-9xl font-bold text-muted-foreground/20">404</h1>
      <p className="mt-4 text-xl font-medium">Page not found</p>
      <p className="mt-2 text-muted-foreground">
        The page you're looking for doesn't exist or has been moved.
      </p>
      <div className="mt-6 flex gap-4">
        <Link to="/">
          <Button variant="default" className="flex items-center gap-2">
            <Home className="w-4 h-4" />
            Go Home
          </Button>
        </Link>
        <Button variant="outline" className="flex items-center gap-2">
          <RotateCcw className="w-4 h-4" />
          Refresh
        </Button>
      </div>
    </div>
  );
}
