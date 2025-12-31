# Hướng dẫn Deploy NoteMe lên Render.com

## 🚀 Quick Start

### Bước 1: Chuẩn bị Code

Đảm bảo code đã được commit và push lên GitHub:
```bash
git add .
git commit -m "Ready for Render deployment"
git push origin main
```

### Bước 2: Tạo Service trên Render

1. **Đăng ký/Login:** https://render.com
2. **New → Web Service**
3. **Connect GitHub repository:**
   - Chọn repo `noteme`
   - Chọn branch `main`

### Bước 3: Cấu hình Service

Render sẽ tự detect từ `render.yaml`, hoặc bạn có thể set manual:

**Basic Settings:**
- **Name:** `noteme-backend`
- **Environment:** `Go`
- **Region:** Chọn gần nhất (Singapore cho Việt Nam)
- **Branch:** `main`
- **Root Directory:** (để trống)

**Build & Deploy:**
- **Build Command:** `go mod download && go build -o server cmd/server/main.go`
- **Start Command:** `./server`
- **Plan:** `Free` (hoặc Starter nếu muốn không sleep)

**Advanced Settings:**
- **Health Check Path:** `/health`
- **Auto-Deploy:** `Yes` (tự động deploy khi có commit mới)

### Bước 4: Set Environment Variables

Vào **Environment** tab, thêm các biến:

| Key | Value | Required |
|-----|-------|----------|
| `FPT_AI_API_KEY` | Your FPT.AI API key | ✅ Yes |
| `FPT_AI_STT_URL` | `https://api.fpt.ai/hmi/asr/v1` | ❌ No (có default) |
| `OPENAI_API_KEY` | Your OpenAI API key | ✅ Yes |
| `GIN_MODE` | `release` | ❌ No |
| `PORT` | (Render tự set) | ❌ No |

**Lưu ý:** 
- Không commit `.env` file
- Set trực tiếp trên Render dashboard

### Bước 5: Deploy

1. Click **Create Web Service**
2. Render sẽ tự động:
   - Install Go 1.21.13 (từ go.mod)
   - Run `go mod download`
   - Build application
   - Start server
3. Chờ build xong (~2-3 phút)
4. Lấy URL từ dashboard (ví dụ: `https://noteme-backend.onrender.com`)

---

## 📋 Kiểm tra Deployment

### 1. Health Check
```bash
curl https://your-app.onrender.com/health
```

Expected response:
```json
{
  "success": true,
  "data": {
    "status": "ok",
    "service": "noteme-backend"
  }
}
```

### 2. Test Upload
```bash
curl -X POST https://your-app.onrender.com/api/v1/recordings \
  -F "audio_file=@test.m4a"
```

---

## ⚠️ Lưu ý Quan Trọng

### Free Tier Limitations

1. **Sleep Mode:**
   - Service sẽ sleep sau 15 phút không có traffic
   - Request đầu tiên sau khi sleep sẽ mất ~30-60 giây để wake up
   - **Giải pháp:** Dùng cron job để ping service mỗi 10 phút, hoặc upgrade lên Starter plan ($7/tháng)

2. **File Storage:**
   - Uploads folder là ephemeral (mất khi restart)
   - **Giải pháp:** Dùng external storage (S3, Cloudinary) cho production

3. **Resource Limits:**
   - 512MB RAM
   - 0.5 CPU
   - Đủ cho MVP nhưng có thể chậm khi xử lý nhiều requests

### Production Recommendations

1. **Upgrade to Starter Plan ($7/tháng):**
   - Không sleep
   - 512MB RAM
   - Better performance

2. **Use External Storage:**
   - AWS S3
   - Cloudinary
   - Google Cloud Storage

3. **Add Monitoring:**
   - Render có built-in logs
   - Có thể tích hợp Sentry cho error tracking

---

## 🔧 Troubleshooting

### Build Failed

**Lỗi:** `go: module noteme: Get ... 410 Gone`
- **Giải pháp:** Đảm bảo `go.mod` đúng version (1.21.13)

**Lỗi:** `cannot find package`
- **Giải pháp:** Chạy `go mod tidy` local và commit lại

### Service Won't Start

**Lỗi:** `port already in use`
- **Giải pháp:** Đảm bảo code dùng `PORT` env var (đã có sẵn)

**Lỗi:** `FPT_AI_API_KEY is required`
- **Giải pháp:** Check environment variables trên Render dashboard

### Service Sleeps Too Often

**Giải pháp:**
1. Upgrade lên Starter plan
2. Hoặc setup cron job ping service:
   ```bash
   # Crontab (chạy mỗi 10 phút)
   */10 * * * * curl https://your-app.onrender.com/health
   ```

---

## 📊 Monitoring

### View Logs
1. Vào Render dashboard
2. Click vào service
3. Tab **Logs** → Xem real-time logs

### Metrics
- Render cung cấp basic metrics:
  - CPU usage
  - Memory usage
  - Request count

---

## 🔄 Auto-Deploy

Render tự động deploy khi:
- Có commit mới lên branch đã connect
- Manual trigger từ dashboard

**Disable auto-deploy:**
- Settings → Auto-Deploy → Disable

---

## 🎯 Next Steps

Sau khi deploy thành công:

1. **Test API:**
   - Health check
   - Upload audio
   - Process recording
   - Analyze transcript

2. **Update Mobile App:**
   - Thay đổi API base URL
   - Test integration

3. **Monitor:**
   - Check logs thường xuyên
   - Monitor error rate
   - Check API response time

---

## 📝 Checklist

- [ ] Code đã push lên GitHub
- [ ] Tạo Web Service trên Render
- [ ] Set environment variables
- [ ] Build thành công
- [ ] Health check pass
- [ ] Test upload audio
- [ ] Test process recording
- [ ] Test analyze
- [ ] Update mobile app với new URL
- [ ] Monitor logs

---

## 🔗 Useful Links

- Render Dashboard: https://dashboard.render.com
- Render Docs: https://render.com/docs
- Go on Render: https://render.com/docs/go

