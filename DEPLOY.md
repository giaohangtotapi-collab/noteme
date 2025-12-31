# Hướng dẫn Deploy NoteMe Backend (Free Tier)

## 🎯 Các Platform Free Tier Phù Hợp

### 1. **Railway.app** ⭐ (Khuyến nghị)

**Ưu điểm:**
- Free tier: $5 credit/tháng (đủ cho MVP)
- Hỗ trợ Go tốt
- Auto-deploy từ GitHub
- Environment variables dễ cấu hình
- File storage (ephemeral - mất khi restart)
- SSL tự động

**Setup:**
1. Đăng ký tại: https://railway.app
2. Connect GitHub repo
3. New Project → Deploy from GitHub
4. Chọn repo → Railway tự detect Go
5. Set environment variables:
   - `FPT_AI_API_KEY`
   - `FPT_AI_STT_URL`
   - `OPENAI_API_KEY`
   - `PORT` (Railway tự set, không cần)

**Lưu ý:** File uploads sẽ mất khi restart. Cần dùng external storage (S3, Cloudinary) cho production.

---

### 2. **Render.com** ⭐

**Ưu điểm:**
- Free tier: 750 giờ/tháng
- Hỗ trợ Go
- Auto-deploy từ GitHub
- SSL tự động
- Environment variables

**Nhược điểm:**
- Sleep sau 15 phút không có traffic (free tier)
- File storage ephemeral

**Setup:**
1. Đăng ký tại: https://render.com
2. New → Web Service
3. Connect GitHub repo
4. Settings:
   - Build Command: `go build -o server cmd/server/main.go`
   - Start Command: `./server`
   - Environment: Go
5. Set environment variables

---

### 3. **Fly.io** ⭐

**Ưu điểm:**
- Free tier: 3 shared-cpu VMs
- Hỗ trợ Go tốt
- Global edge network
- Persistent volumes (có thể dùng cho uploads)

**Setup:**
1. Install Fly CLI: `curl -L https://fly.io/install.sh | sh`
2. Login: `fly auth login`
3. Init: `fly launch`
4. Set secrets: `fly secrets set FPT_AI_API_KEY=xxx OPENAI_API_KEY=xxx`

**File:** Tạo `fly.toml` (Fly sẽ tự generate)

---

### 4. **Google Cloud Run** (Free Tier)

**Ưu điểm:**
- Free tier: 2 triệu requests/tháng
- Pay-as-you-go sau free tier
- Auto-scaling
- Container-based

**Nhược điểm:**
- Cần Dockerfile
- Setup phức tạp hơn

**Setup:**
1. Tạo Dockerfile
2. Build: `gcloud builds submit --tag gcr.io/PROJECT_ID/noteme`
3. Deploy: `gcloud run deploy`

---

### 5. **DigitalOcean App Platform** (Free Trial)

**Ưu điểm:**
- $200 credit free trial (60 ngày)
- Hỗ trợ Go
- Auto-deploy

**Nhược điểm:**
- Chỉ free trial, không phải free tier vĩnh viễn

---

## 📋 Checklist Trước Khi Deploy

### 1. Chuẩn bị Code
- [ ] Code đã test local
- [ ] Environment variables đã document
- [ ] Port động (dùng `PORT` env var)
- [ ] Logging phù hợp

### 2. File Storage
- [ ] Quyết định: ephemeral (mất khi restart) hay persistent
- [ ] Nếu cần persistent: setup S3/Cloudinary/Google Cloud Storage

### 3. Environment Variables Cần Set
```
FPT_AI_API_KEY=your_key
FPT_AI_STT_URL=https://api.fpt.ai/hmi/asr/v1
OPENAI_API_KEY=your_key
PORT=8080 (hoặc để platform tự set)
GIN_MODE=release
```

---

## 🚀 Quick Start: Railway (Khuyến nghị)

### Bước 1: Chuẩn bị Repo
```bash
# Đảm bảo code đã commit và push lên GitHub
git add .
git commit -m "Ready for deployment"
git push origin main
```

### Bước 2: Deploy trên Railway
1. Truy cập: https://railway.app
2. Login với GitHub
3. New Project → Deploy from GitHub
4. Chọn repo `noteme`
5. Railway tự detect Go và build

### Bước 3: Set Environment Variables
1. Vào project → Variables
2. Add từng variable:
   - `FPT_AI_API_KEY`
   - `FPT_AI_STT_URL`
   - `OPENAI_API_KEY`

### Bước 4: Deploy
- Railway tự động deploy
- Lấy URL từ Settings → Domains

---

## 🐳 Dockerfile (Nếu cần)

Tạo `Dockerfile` nếu deploy lên Cloud Run hoặc tự host:

```dockerfile
FROM golang:1.21-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o server cmd/server/main.go

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/server .
COPY --from=builder /app/uploads ./uploads

EXPOSE 8080
CMD ["./server"]
```

---

## 📝 Lưu Ý Quan Trọng

### File Storage
- **Free tier thường không có persistent storage**
- Uploads sẽ mất khi container restart
- Giải pháp:
  1. Dùng external storage (S3, Cloudinary)
  2. Hoặc chấp nhận mất file (cho MVP)

### Environment Variables
- **KHÔNG commit `.env` vào Git**
- Set trên platform dashboard
- Railway/Render có UI để set dễ dàng

### Port
- Platform thường tự set `PORT` env var
- Code đã hỗ trợ: `r.Run(":" + cfg.Port)`

### CORS
- Code đã set CORS cho mobile app
- Có thể cần điều chỉnh `Access-Control-Allow-Origin` cho production

---

## 🔗 Links Hữu Ích

- Railway: https://railway.app
- Render: https://render.com
- Fly.io: https://fly.io
- Google Cloud Run: https://cloud.google.com/run
- DigitalOcean: https://www.digitalocean.com

---

## 💡 Khuyến nghị

**Cho MVP:**
1. **Railway** - Dễ nhất, free tier tốt
2. **Render** - Nếu Railway hết credit

**Cho Production:**
- Railway/Render paid plan
- Hoặc VPS (DigitalOcean, Linode) ~$5/tháng

