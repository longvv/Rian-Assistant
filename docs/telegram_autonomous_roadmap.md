# Product Vision: PicoClaw Autonomous Personal Assistant (Telegram Integration)

## 🌟 Tầm nhìn (Product Vision)

Chuyển đổi PicoClaw Telegram bot từ một **"Công cụ phản hồi thụ động"** (nhận lệnh -> xử lý -> trả kết quả) thành một **"Trợ lý cá nhân tự động 24/7"** (Autonomous Background Agent). Trợ lý này có khả năng tự quan sát, tự động tổng hợp thông tin, theo dõi thói quen người dùng và chủ động báo cáo hoặc xin phép thực thi các tác vụ quản trị thông qua Telegram, tất cả với mức tiêu hao tài nguyên tối thiểu (<10MB RAM) và chi phí LLM bằng $0 (sử dụng OpenRouter free tier).

## 🏗️ Kiến trúc Cốt lõi: 3-Tier Micro-Agent Orchestration (Passive Trigger)

Mô hình "Orchestration thụ động" giúp bảo toàn đặc tính tiết kiệm RAM của PicoClaw. Các agent chỉ được kích hoạt (nhảy vào RAM) khi thực sự cần thiết, thay vì chạy nền liên tục.

### 1. The Gateway / Intent Router (Người gác cổng)

- **Model:** `meta-llama/llama-3.2-3b-instruct:free` (Tối ưu tốc độ).
- **Vai trò:** Tiếp nhận 100% tin nhắn Telegram. Dùng Prompt siêu nhẹ (<50 tokens) để phân tích ý định (Intent Parser).
- **Hành động:**
  - _Intent đơn giản_ (Chitchat, lệnh cơ bản): Xử lý và trả lời ngay.
  - _Intent phức tạp_ (Cần gọi nhiều tool, lập kế hoạch): Kích hoạt (Trigger) Orchestrator.

### 2. The Orchestrator / Synthesizer (Tổng công trình sư)

- **Model:** `arcee-ai/trinity-large-preview:free` (Tối ưu IQ/Tuân thủ nguyên tắc).
- **Vai trò:** Não bộ chính yếu, thức dậy khi Router báo động.
- **Hành động:**
  - _Task Decomposition:_ Chia yêu cầu phức tạp thành các Sub-tasks.
  - _Delegation:_ Dùng lệnh `spawn` (qua Goroutines) để gọi các Worker chạy song song.
  - _Synthesis:_ Tổng hợp kết quả từ các Worker, viết báo cáo hoàn chỉnh dưới dạng Markdown/Text và gửi về Telegram.

### 3. The Worker / Tool Executor (Kỹ sư tuyến đầu)

- **Model:** `nvidia/nemotron-3-nano-30b-a3b:free` (Tối ưu sử dụng Tool - Function Calling).
- **Vai trò:** Thực thi các tác vụ cụ thể do Orchestrator giao phó (Không giao tiếp với User).
- **Hành động:** Gọi Web Search, đọc file hệ thống, chạy Bash Script, gom data thô (JSON/Raw Text) gửi ngược lại cho Orchestrator.

---

## 🚀 Lộ trình Triển khai (Implementation Roadmap)

### Phase 1: Core Architecture & Proactive Foundation (Foundation)

_Mục tiêu: Đặt nền móng cho kiến trúc 3-tier và khả năng chủ động (Proactive)._

- [ ] **Tích hợp Intent Router vào Telegram Channel (`pkg/channels/telegram.go`):**
  - Thay thế hệ thống bắt lệnh cứng nhắc (`/command`) bằng luồng đi qua Gateway model (`Llama-3.2-3b`).
- [ ] **Xây dựng Orchestrator Module:**
  - Tạo cơ chế truyền nhận tín hiệu giữa Router và Orchestrator. Cấu hình context isolation (cô lập bối cảnh) để tiết kiệm bộ nhớ.
- [ ] **Worker Spawning với Goroutines:**
  - Mở rộng lệnh `spawn` hiện tại để hỗ trợ gọi Worker (`Nemotron`) thực thi song song (Parallel execution) các tool (vd: Web Search nhiều nguồn cùng lúc).
- [ ] **Kết nối Heartbeat với Telegram (Proactive Briefs):**
  - Cho phép `HEARTBEAT.md` gửi báo cáo định kỳ trực tiếp vào Telegram (VD: Báo cáo `/news` mỗi sáng) mà không cần User gõ lệnh.

### Phase 2: UX Enhancements & Context Awareness (Experience)

_Mục tiêu: Nâng cao trải nghiệm người dùng trên Telegram và cá nhân hóa._

- [ ] **Interactive Inline Keyboards:**
  - Nâng cấp cách hiển thị bằng các nút bấm (Buttons). Báo cáo tin tức/cảnh báo sẽ kèm theo các nút hành động nhanh (Ví dụ: `[Tóm tắt ngắn]`, `[Đọc bản full]`, `[Tìm hiểu sâu]`).
- [ ] **Context-Aware Memory (Session Management):**
  - Nâng cấp `MEMORY.md` để phân luồng bộ nhớ theo Telegram `ChatID`. Bot sẽ nhớ bối cảnh riêng biệt khi chat trong Group vs chat Private.
- [ ] **Habit Tracker (Nhận diện thói quen):**
  - Agent tự động phân tích lịch sử chat để cập nhật `USER.md`, phát hiện các tác vụ lặp đi lặp lại và đề xuất tự động hóa chúng.

### Phase 3: Total Automation & Event-Driven (Autonomy)

_Mục tiêu: Biến PicoClaw thành hệ thống tự động hoàn toàn._

- [ ] **Event-Driven Webhooks:**
  - Tạo endpoint để Webhook từ các hệ thống khác (GitLab, Monitor, Calendar) có thể trigger PicoClaw. Orchestrator sẽ lấy data đó, phân tích và chủ động báo cáo cho Admin qua Telegram.
- [ ] **Interactive Approval Workflows (Human-in-the-loop):**
  - Khi hệ thống phát hiện lỗi (qua Auto-healing) hoặc cần cấu hình rủi ro cao, Bot chủ động gửi cảnh báo kèm giải pháp khắc phục. User chỉ cần bấm nút `[Approve]` trên Telegram, Worker sẽ tự động thực thi.

## 📝 User Review Required

Bạn nghĩ sao về **Implementation roadmap** này? Nếu bạn đồng ý với tầm nhìn và các phase triển khai, chúng ta có thể bắt đầu bằng việc đi vào **Phase 1: Xây dựng Core Architecture (Gateway & Orchestrator)** trước. Báo lại cho tôi sự đồng thuận của bạn để tôi bắt đầu tạo các branch hoặc cập nhật code nhé!
