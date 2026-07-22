export default function GlobalLoading() {
  return (
    <main className="wrap global-loading" aria-label="Đang tải trang" aria-busy="true">
      <div className="global-loading-heading">
        <div className="skeleton global-loading-kicker" />
        <div className="skeleton global-loading-title" />
        <div className="skeleton global-loading-subtitle" />
      </div>
      <div className="global-loading-grid">
        <div className="skeleton global-loading-lead" />
        <div className="global-loading-stack">
          <div className="skeleton global-loading-card" />
          <div className="skeleton global-loading-card" />
          <div className="skeleton global-loading-card" />
        </div>
      </div>
    </main>
  );
}
