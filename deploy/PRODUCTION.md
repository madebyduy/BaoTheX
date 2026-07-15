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
