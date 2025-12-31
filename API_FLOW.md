# API Flow - NoteMe Mobile App Integration

## 📱 Luồng Xử Lý Hoàn Chỉnh

### Flow Diagram

```
User bấm nút → Ghi âm (30s) → Upload → Process → Analyze → Hiển thị kết quả
```

---

## 🔄 Chi Tiết Từng Bước

### **BƯỚC 1: Ghi âm (Local - App)**

**App thực hiện:**
- User bấm nút → Bắt đầu ghi âm
- Tự động dừng sau 30 giây (hoặc user dừng thủ công)
- Lưu file audio tạm trên device (format: m4a, mp3, wav)

**Không cần gọi API ở bước này**

---

### **BƯỚC 2: Upload Audio File**

**API Call:**
```http
POST /api/v1/recordings
Content-Type: multipart/form-data
```

**Request:**
```javascript
const formData = new FormData();
formData.append('audio_file', audioFile); // File object từ recording

const response = await fetch('https://your-api.com/api/v1/recordings', {
  method: 'POST',
  body: formData,
  headers: {
    // Không set Content-Type, browser tự set với boundary
  }
});
```

**Response (Success - 200):**
```json
{
  "success": true,
  "data": {
    "recording_id": "rec_1767075531263315800",
    "status": "uploaded"
  }
}
```

**Response (Error - 400):**
```json
{
  "success": false,
  "error": "unsupported audio format. Supported: m4a, mp3, wav, aac, ogg"
}
```

**Lưu `recording_id` để dùng cho các bước sau**

---

### **BƯỚC 3: Process Recording (STT + Clean)**

**API Call:**
```http
POST /api/v1/process/:recording_id
```

**Request:**
```javascript
const recordingId = "rec_1767075531263315800";

const response = await fetch(
  `https://your-api.com/api/v1/process/${recordingId}`,
  {
    method: 'POST',
  }
);
```

**Response (Success - 200):**
```json
{
  "success": true,
  "data": {
    "recording_id": "rec_1767075531263315800",
    "status": "processed",
    "language": "vi",
    "transcript": "Nội dung đã được chuyển đổi và làm sạch...",
    "confidence": 0.95
  }
}
```

**Response (Error - 400):**
```json
{
  "success": false,
  "error": "no speech detected in audio"
}
```

**Lưu ý:**
- API này sẽ tự động:
  1. Gọi FPT.AI để chuyển audio → transcript
  2. Gọi OpenAI để làm sạch transcript (fix lỗi nhận dạng)
- Thời gian xử lý: ~10-20 giây
- Có thể polling status nếu muốn async

---

### **BƯỚC 4: Analyze với AI (Optional - Nếu cần insights)**

**API Call:**
```http
POST /api/v1/ai/analyze/:recording_id
```

**Request:**
```javascript
const response = await fetch(
  `https://your-api.com/api/v1/ai/analyze/${recordingId}`,
  {
    method: 'POST',
  }
);
```

**Response (Success - 200):**
```json
{
  "success": true,
  "data": {
    "recording_id": "rec_1767075531263315800",
    "context": "meeting",
    "summary": [
      "Khách hàng yêu cầu dự án BĐS nghỉ dưỡng",
      "Ngân sách khoảng 50 tỷ"
    ],
    "action_items": [
      "Chuẩn bị proposal chi tiết",
      "Gửi báo giá trước thứ Sáu"
    ],
    "key_points": [
      "Ngân sách: 50 tỷ",
      "Deadline: Thứ Sáu"
    ],
    "zalo_brief": "- Khách yêu cầu dự án BĐS\n- Ngân sách 50 tỷ\n- Deadline thứ Sáu"
  }
}
```

**Lưu ý:**
- Chỉ gọi khi cần insights (summary, action items, key points)
- Nếu chỉ cần transcript, bỏ qua bước này
- Thời gian xử lý: ~5-10 giây

---

## 🎯 Flow Tối Ưu Cho App

### **Option 1: Sync Flow (Đơn giản)**

```javascript
async function processRecording(audioFile) {
  try {
    // 1. Upload
    const uploadResponse = await uploadAudio(audioFile);
    const { recording_id } = uploadResponse.data;
    
    // 2. Process (chờ kết quả)
    const processResponse = await processRecording(recording_id);
    const { transcript, status } = processResponse.data;
    
    // 3. Analyze (optional)
    const analysisResponse = await analyzeRecording(recording_id);
    const { summary, action_items, key_points } = analysisResponse.data;
    
    return {
      transcript,
      analysis: {
        summary,
        action_items,
        key_points
      }
    };
  } catch (error) {
    console.error('Error:', error);
    throw error;
  }
}
```

**Ưu điểm:** Đơn giản, dễ implement  
**Nhược điểm:** User phải chờ ~20-30 giây

---

### **Option 2: Async Flow với Polling (Tốt hơn UX)**

```javascript
async function processRecordingAsync(audioFile) {
  try {
    // 1. Upload
    const uploadResponse = await uploadAudio(audioFile);
    const { recording_id } = uploadResponse.data;
    
    // 2. Process (async)
    await processRecording(recording_id);
    
    // 3. Poll status
    const status = await pollStatus(recording_id);
    
    if (status === 'processed') {
      // 4. Get transcript
      const transcript = await getTranscript(recording_id);
      
      // 5. Analyze (background)
      analyzeRecording(recording_id).then(analysis => {
        // Update UI khi có kết quả
        updateUI(analysis);
      });
      
      return { transcript };
    }
  } catch (error) {
    console.error('Error:', error);
    throw error;
  }
}

async function pollStatus(recordingId, maxAttempts = 30) {
  for (let i = 0; i < maxAttempts; i++) {
    const response = await fetch(
      `https://your-api.com/api/v1/recordings/${recordingId}/status`
    );
    const { data } = await response.json();
    
    if (data.status === 'processed' || data.status === 'failed') {
      return data.status;
    }
    
    // Chờ 1 giây trước khi poll lại
    await new Promise(resolve => setTimeout(resolve, 1000));
  }
  
  throw new Error('Timeout waiting for processing');
}
```

**Ưu điểm:** UX tốt hơn, có thể hiển thị progress  
**Nhược điểm:** Code phức tạp hơn

---

## 📋 API Endpoints Summary

### **1. Upload Audio**
```
POST /api/v1/recordings
Body: multipart/form-data (audio_file)
Response: { recording_id, status }
```

### **2. Process Recording**
```
POST /api/v1/process/:recording_id
Response: { transcript, confidence, status }
```

### **3. Get Recording Status**
```
GET /api/v1/recordings/:recording_id/status
Response: { recording_id, status }
```

### **4. Get Recording Info**
```
GET /api/v1/recordings/:recording_id
Response: { transcript, confidence, status, created_at }
```

### **5. Analyze Recording**
```
POST /api/v1/ai/analyze/:recording_id
Response: { context, summary, action_items, key_points, zalo_brief }
```

### **6. Get Analysis**
```
GET /api/v1/ai/analyze/:recording_id
Response: { context, summary, action_items, key_points, zalo_brief }
```

### **7. Health Check**
```
GET /health
Response: { status: "ok", service: "noteme-backend" }
```

---

## 🔄 Recommended Flow cho MVP

### **Flow 1: Chỉ cần Transcript (Nhanh nhất)**

```
1. Upload audio → Get recording_id
2. Process → Get transcript
3. Hiển thị transcript cho user
```

**Thời gian:** ~15-20 giây

---

### **Flow 2: Full Analysis (Đầy đủ nhất)**

```
1. Upload audio → Get recording_id
2. Process → Get transcript (hiển thị ngay)
3. Analyze → Get insights (hiển thị sau)
4. Hiển thị: Transcript + Summary + Action Items + Key Points
```

**Thời gian:** ~25-30 giây

---

### **Flow 3: Background Processing (UX tốt nhất)**

```
1. Upload audio → Get recording_id
2. Process (background) → Show loading
3. Khi có transcript → Hiển thị transcript
4. Analyze (background) → Update UI khi có insights
```

**Thời gian:** User thấy transcript sau ~15s, insights sau ~25s

---

## ⚠️ Error Handling

### **Common Errors:**

1. **400 - Bad Request**
   - Unsupported format
   - File too large
   - Missing recording_id

2. **404 - Not Found**
   - Recording không tồn tại
   - Analysis chưa có

3. **500 - Internal Server Error**
   - STT failed
   - AI analysis failed
   - Server error

### **Error Response Format:**
```json
{
  "success": false,
  "error": "Error message here"
}
```

---

## 💡 Best Practices

### **1. Retry Logic**
```javascript
async function retryRequest(fn, maxRetries = 3) {
  for (let i = 0; i < maxRetries; i++) {
    try {
      return await fn();
    } catch (error) {
      if (i === maxRetries - 1) throw error;
      await new Promise(resolve => setTimeout(resolve, 1000 * (i + 1)));
    }
  }
}
```

### **2. Timeout Handling**
```javascript
const controller = new AbortController();
const timeoutId = setTimeout(() => controller.abort(), 30000); // 30s timeout

try {
  const response = await fetch(url, {
    signal: controller.signal
  });
  clearTimeout(timeoutId);
} catch (error) {
  if (error.name === 'AbortError') {
    // Handle timeout
  }
}
```

### **3. Progress Indicator**
- Show "Đang xử lý..." khi process
- Show "Đang phân tích..." khi analyze
- Update UI khi có từng phần kết quả

---

## 📱 Example: React Native / Flutter

### **React Native Example:**
```javascript
import axios from 'axios';

const API_BASE_URL = 'https://your-api.com/api/v1';

class NoteMeAPI {
  static async uploadAudio(audioFile) {
    const formData = new FormData();
    formData.append('audio_file', {
      uri: audioFile.uri,
      type: 'audio/m4a',
      name: 'recording.m4a',
    });
    
    const response = await axios.post(
      `${API_BASE_URL}/recordings`,
      formData,
      {
        headers: {
          'Content-Type': 'multipart/form-data',
        },
      }
    );
    
    return response.data;
  }
  
  static async processRecording(recordingId) {
    const response = await axios.post(
      `${API_BASE_URL}/process/${recordingId}`
    );
    return response.data;
  }
  
  static async analyzeRecording(recordingId) {
    const response = await axios.post(
      `${API_BASE_URL}/ai/analyze/${recordingId}`
    );
    return response.data;
  }
}
```

---

## 🎯 Quick Reference

| Bước | API | Method | Khi nào gọi |
|------|-----|--------|-------------|
| Upload | `/api/v1/recordings` | POST | Sau khi ghi âm xong |
| Process | `/api/v1/process/:id` | POST | Ngay sau upload |
| Get Status | `/api/v1/recordings/:id/status` | GET | Nếu dùng async flow |
| Get Transcript | `/api/v1/recordings/:id` | GET | Khi cần transcript |
| Analyze | `/api/v1/ai/analyze/:id` | POST | Khi cần insights |
| Get Analysis | `/api/v1/ai/analyze/:id` | GET | Khi cần lấy lại analysis |

---

## ✅ Checklist Implementation

- [ ] Setup API base URL
- [ ] Implement upload audio function
- [ ] Implement process recording function
- [ ] Implement analyze function
- [ ] Add error handling
- [ ] Add loading states
- [ ] Add retry logic
- [ ] Test với audio thật
- [ ] Optimize UX (async flow nếu cần)

