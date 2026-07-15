export default function LoadingArticle() {
  return (
    <main className="wrap article-page article-loading" aria-label="Đang tải bài viết">
      <div className="skeleton skeleton-back" />
      <div className="skeleton skeleton-kicker" />
      <div className="skeleton skeleton-title" />
      <div className="skeleton skeleton-title short" />
      <div className="skeleton skeleton-lede" />
      <div className="skeleton skeleton-hero" />
      <div className="loading-columns">
        <div className="skeleton skeleton-aside" />
        <div className="skeleton skeleton-copy" />
      </div>
    </main>
  );
}
