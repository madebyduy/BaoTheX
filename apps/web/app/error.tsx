"use client";

import { useEffect } from "react";
import Link from "next/link";

export default function Error({
  error,
  reset,
}: {
  error: Error & { digest?: string };
  reset: () => void;
}) {
  useEffect(() => {
    // Surface the error to the console (and Sentry, if wired) for diagnosis.
    console.error(error);
  }, [error]);

  return (
    <main className="wrap error-page">
      <span className="error-code">Ối</span>
      <h1>Đã có lỗi xảy ra</h1>
      <p>
        Xin lỗi vì sự bất tiện. Bạn có thể thử tải lại trang; nếu lỗi vẫn tiếp diễn, hãy quay lại
        sau ít phút.
      </p>
      <div className="error-actions">
        <button className="btn ember" type="button" onClick={reset}>
          Thử lại
        </button>
        <Link className="btn light" href="/">
          Về trang chủ
        </Link>
      </div>
    </main>
  );
}
