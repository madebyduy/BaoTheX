# Triển khai BaoTheX với Supabase

Kiến trúc production đề xuất: một VPS Ubuntu chạy `web`, `api`, `worker` và
`Caddy`; Supabase tiếp tục giữ PostgreSQL. Caddy tự cấp và gia hạn HTTPS.

## 1. Chuẩn bị

- Một domain đã trỏ bản ghi `A` về IP của VPS.
- Docker Engine và Docker Compose plugin trên VPS.
- Repository BaoTheX đã được đẩy lên GitHub.
- Xoay mới toàn bộ API key từng xuất hiện trong chat trước khi đưa production.

## 2. Cấu hình bí mật

```bash
git clone https://github.com/madebyduy/BaoTheX.git
cd BaoTheX
cp .env.example .env
chmod 600 .env
```

Tối thiểu phải đặt: `DOMAIN`, `DATABASE_URL` (Supabase Session Pooler),
`SESSION_SECRET`, `PUBLIC_BASE_URL=https://<domain>`, `CORS_ORIGINS`, Gemini,
YouTube, Telegram và SePay. Không commit `.env`.

Tạo tài khoản quản trị trước lần mở báo:

```bash
set -a
. ./.env
set +a
go run ./tools/seed-admin
```

## 3. Khởi chạy

```bash
docker compose -f deploy/docker-compose.prod.yml --env-file .env up -d --build
docker compose -f deploy/docker-compose.prod.yml ps
docker compose -f deploy/docker-compose.prod.yml logs -f api worker
```

Kiểm tra `https://<domain>/healthz`, trang chủ và `/admin`.

`/healthz` chỉ xác nhận tiến trình API đang sống; `/readyz` xác nhận kết nối
database và chỉ được dùng từ mạng nội bộ. API còn cung cấp `/metrics` theo định
dạng Prometheus trên cổng nội bộ `8080` (Caddy không công khai route này), gồm
tổng request, 5xx, request đang xử lý và tổng thời gian xử lý.

Docker build production bật kiểm tra cấu hình nghiêm ngặt: cả
`NEXT_PUBLIC_API_URL` và `NEXT_PUBLIC_SITE_URL` phải tồn tại và dùng HTTPS. Build
sẽ dừng thay vì âm thầm đóng gói URL localhost.

## 4. Telegram production

Tắt polling trong production và đặt webhook về
`https://<domain>/api/v1/telegram/webhook`. Dùng secret webhook riêng, không
dùng lại mật khẩu database hoặc mật khẩu admin.

## 5. Cập nhật phiên bản

```bash
git pull --ff-only origin main
docker compose -f deploy/docker-compose.prod.yml --env-file .env up -d --build
```

Audio/video nằm trong Docker volume `media`; sao lưu volume này cùng Supabase.

## 6. Giám sát và cảnh báo tối thiểu

- Uptime check mỗi phút: `/healthz`, trang chủ và một trang nội dung thật.
- Alert nếu `/readyz` lỗi, tỷ lệ 5xx vượt 2% trong 5 phút, hoặc p95 API vượt 1s.
- Scrape `http://api:8080/metrics` từ agent trong Docker network; không mở cổng
  metrics ra Internet.
- Thu log JSON của `api` và `worker`, cảnh báo các chuỗi `dead`, `panic`,
  `webhook handler failed`, `db connect failed` và queue tăng liên tục.
- Đặt `SENTRY_DSN` sau khi đã cài/cấu hình tài khoản Sentry; khi chưa có DSN,
  integration frontend là no-op và không gửi dữ liệu ra ngoài.
- Chạy `npm audit --omit=dev` và `govulncheck ./...` trong CI theo lịch hằng tuần.

## 7. Sao lưu và diễn tập khôi phục

Không coi backup là hợp lệ cho tới khi restore thử thành công. Hằng ngày sao lưu
database Supabase và volume `media`; giữ ít nhất 7 bản ngày và 4 bản tuần. Mỗi
tháng khôi phục vào database/volume tạm, chạy migration, `/readyz`, đăng nhập,
mở một bài và phát một audio brief. Ghi lại RPO/RTO thực tế và thời điểm bản sao
gần nhất đã được kiểm chứng.

## 8. Checklist trước khi phát hành

```bash
go test ./...
govulncheck ./...
cd apps/web
npm ci
npm run lint
npm run build
npm audit --omit=dev
npm run test:e2e
```

Sau deploy, kiểm tra CSP/security headers, đăng ký/đăng nhập/logout, quyền admin,
follow/save, checkout và một IPN sandbox. Không dùng giao dịch thật để smoke test.
