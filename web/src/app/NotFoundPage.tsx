import { Link } from "react-router-dom";

export function NotFoundPage() {
  return (
    <div className="empty-state">
      <h2>Page not found</h2>
      <p>That page doesn't exist, or the link is out of date.</p>
      <Link to="/overview" className="btn btn-primary">
        Back to Overview
      </Link>
    </div>
  );
}
