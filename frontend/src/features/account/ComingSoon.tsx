import { useSetBreadcrumbs } from "../../app/BreadcrumbContext";
import { EmptyState } from "../../components/EmptyState";
import "./Account.css";

/** A named place for something that does not exist yet.
 *
 *  The account menu lists these, so they need somewhere to land. The
 *  screen says plainly that there is nothing behind it rather than
 *  showing an empty table that looks like a bug or, worse, like data.
 */
export function ComingSoon({ title, description }: { title: string; description: string }) {
  useSetBreadcrumbs([{ label: title }]);
  return (
    <div className="account-page">
      <header>
        <h1>{title}</h1>
      </header>
      <EmptyState title="Not built yet" description={description} />
    </div>
  );
}
