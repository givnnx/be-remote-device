# be-remote-device

Backend REST API untuk manajemen, monitoring, dan pengiriman perintah jarak jauh ke perangkat IoT / edge device (*remote devices*) yang dilengkapi dengan **sistem Action Log / Audit Trail yang disimpan langsung di Database (PostgreSQL)**.

## Fitur Utama

- **Device Management**: Registrasi, melihat daftar, detail, update, dan hapus device.
- **Heartbeat & Status Monitoring**: Memperbarui status ketersediaan dan `last_seen` perangkat secara realtime.
- **Remote Command Dispatch**: Mengirim perintah (*payload*) ke perangkat dan memperbarui hasil eksekusi (*status & result*).
- **Telemetry Ingestion**: Menerima metrik performa perangkat (CPU, memory, disk, baterai, suhu).
- **Database Action & Audit Logging**:
  - Menyimpan setiap interaksi API ke database (PostgreSQL table `action_logs`).
  - Menangkap informasi pelaku (*actor*), aksi (*action name*), method, path URL, client IP, user agent, status code, durasi eksekusi (*latency*), request body, response snippet, dan error message.
  - Asynchronous background worker untuk penulisan ke DB, sehingga tidak menambah latensi pada response HTTP.
  - Graceful fallback ke in-memory storage jika database sedang offline.
- **Clean Architecture Ready**: Pemisahan jelas antara layer *models*, *repository*, *service*, dan *delivery (HTTP)*.

---

## Struktur Direktori

```text
be-remote-device/
├── cmd/
│   └── api/
│       └── main.go                  # Entrypoint server (DB init, router, graceful shutdown)
├── internal/
│   ├── config/
│   │   └── config.go                # Pengaturan environment & .env loader
│   ├── database/
│   │   └── database.go              # Koneksi DB PostgreSQL & auto-migration table
│   ├── models/
│   │   ├── action_log.go            # Model Action Log / Audit Trail
│   │   ├── command.go               # Model remote command
│   │   ├── device.go                # Model data device & status
│   │   └── telemetry.go             # Model metrik performa device
│   ├── repository/
│   │   ├── action_log_repository.go # Repository Action Log (Postgres & Memory Fallback)
│   │   └── device_repository.go     # Interface repository device
│   ├── service/
│   │   ├── action_log_service.go    # Service Action Log dengan Async Worker Buffer
│   │   └── device_service.go        # Business logic device
│   └── delivery/
│       └── http/
│           ├── audit_middleware.go  # Interceptor otomatis untuk merekam log & error
│           ├── handler.go           # Device HTTP handlers
│           ├── log_handler.go       # Action Log HTTP query handlers
│           └── router.go            # Router & Middleware pipeline
├── pkg/
│   ├── logger/
│   │   └── logger.go                # Structured logger (slog)
│   └── response/
│       └── response.go              # Helper standar format JSON response
├── Dockerfile                       # Multi-stage Docker build
├── docker-compose.yml               # Compose configuration
├── .env.example                     # Contoh file konfigurasi environment
├── go.mod                           # Modul Go & dependencies
└── README.md
```

---

## Cara Menjalankan

### 1. Konfigurasi Database (.env)

Sesuaikan konfigurasi PostgreSQL di file `.env`:

```env
APP_NAME=be-remote-device
APP_ENV=development
APP_PORT=8080
LOG_LEVEL=info

DB_DRIVER=postgres
DB_HOST=localhost
DB_PORT=5432
DB_USER=postgres
DB_PASSWORD=postgres
DB_NAME=remote_device_db
DB_SSLMODE=disable
```

> **Catatan**: Jika database PostgreSQL belum aktif atau belum dibuat, aplikasi akan secara otomatis beralih ke mode *in-memory repository fallback* tanpa menyebabkan server crash.

### 2. Menjalankan Server Lokal

```bash
cd D:\Projects\be-remote-device
go run cmd/api/main.go
```

### 3. Menjalankan Unit Tests

```bash
go test -v ./...
```

### 4. Menggunakan Docker

```bash
docker compose up -d --build
```

---

## Mekanisme Action & Audit Log di Database

Tabel yang dibuat otomatis di database:
```sql
CREATE TABLE IF NOT EXISTS action_logs (
    id VARCHAR(64) PRIMARY KEY,
    timestamp TIMESTAMPTZ NOT NULL,
    actor VARCHAR(255) NOT NULL,
    action VARCHAR(100) NOT NULL,
    method VARCHAR(10) NOT NULL,
    path TEXT NOT NULL,
    client_ip VARCHAR(50) NOT NULL,
    user_agent TEXT,
    status_code INT NOT NULL,
    duration_ms BIGINT NOT NULL,
    request_body TEXT,
    response_body TEXT,
    error_message TEXT,
    status VARCHAR(20) NOT NULL
);
```

### Deteksi Pelaku Request (Actor):
Aplikasi mendeteksi pelaku secara bertingkat:
1. Header `X-User-ID` atau `X-Actor` (contoh: `admin-01`, `operator-john`)
2. Header `X-API-Key` (akan dimaskir untuk keamanan)
3. Header `Authorization`
4. Jika tidak ada header identitas, otomatis menggunakan IP klien (`ip:192.168.x.x`).

---

## Fitur Keamanan, Multi-User & Device Binding

Backend ini dilengkapi dengan arsitektur **Multi-User & Device Scoping**:

1. **User Authentication (Register & Login)**:
   - Pengguna mendaftar secara mandiri melalui `POST /api/v1/auth/register` (password di-hash dengan standar `bcrypt`).
   - Login via `POST /api/v1/auth/login` menghasilkan token JWT berdurasi 7 hari.
   - Seluruh data perangkat (device, command, telemetry) **terikat secara eksklusif ke `user_id` pemilik**. Pengguna lain tidak dapat melihat atau mengendalikan perangkat yang bukan miliknya.

2. **Per-Device API Key (Untuk Laptop & Smartphone)**:
   - Saat perangkat didaftarkan, sistem otomatis menghasilkan `api_key` unik (contoh: `devkey_9b2e...`).
   - Simpan `api_key` ini pada skrip background/daemon di **laptop** atau **smartphone** Anda.
   - Device mengirimkan data (heartbeat, telemetry, hasil perintah) cukup dengan menyertakan header:
     - `X-Device-Key: <devkey_...>`
     - atau query param `?api_key=<devkey_...>`
   - Tidak perlu pusing memikirkan token kedaluwarsa pada background service perangkat Anda.

3. **Master API Key (`MASTER_API_KEY`)**:
   - Berfungsi sebagai emergency key di file `.env` jika diperlukan akses langsung.

> 🛡️ **Audit Keamanan Otomatis**: Jika ada request mencurigakan atau tanpa autentikasi yang sah (`401 Unauthorized`), IP pelaku, path yang dicoba, dan waktu kejadian akan **otomatis tersimpan ke tabel database `action_logs`** sehingga Anda dapat melacak percobaan intrusi.

---

## Dokumentasi API Endpoint

Base URL: `http://localhost:8080`

### 1. Autentikasi User

#### Register Akun Baru
- **`POST /api/v1/auth/register`**
- Body:
```json
{
  "username": "giovanni",
  "email": "giovanni@personal.com",
  "password": "strongpassword123",
  "full_name": "Giovanni Agung"
}
```
- Response:
```json
{
  "success": true,
  "message": "User registered successfully",
  "data": {
    "id": "usr-8a2b3c4d",
    "username": "giovanni",
    "email": "giovanni@personal.com",
    "full_name": "Giovanni Agung",
    "created_at": "2026-09-27T16:10:00Z"
  }
}
```

#### Login User
- **`POST /api/v1/auth/login`**
- Body (dapat menggunakan username atau email):
```json
{
  "username": "giovanni",
  "password": "strongpassword123"
}
```
- Response:
```json
{
  "success": true,
  "message": "Login successful",
  "data": {
    "token": "eyJhbGciOiJIUzI1NiIsIn...",
    "token_type": "Bearer",
    "expires_in": 604800,
    "user": {
      "id": "usr-8a2b3c4d",
      "username": "giovanni",
      "email": "giovanni@personal.com",
      "full_name": "Giovanni Agung"
    }
  }
}
```

#### Profil Saya
- **`GET /api/v1/auth/me`**
- Header: `Authorization: Bearer <JWT_TOKEN>`

---

### 2. Action & Audit Logs

#### Melihat Daftar Action Log (dengan filter & pagination)
- **`GET /api/v1/logs?actor=admin&action=REGISTER_DEVICE&status=SUCCESS&limit=20&offset=0`**
- Query Parameters (opsional):
  - `actor`: Filter berdasarkan nama/ID pelaku
  - `action`: Filter berdasarkan aksi (contoh: `REGISTER_DEVICE`, `SEND_COMMAND`, dll)
  - `status`: `SUCCESS`, `FAILED`, `ERROR`
  - `limit`: Jumlah data per halaman (default 20)
  - `offset`: Offset paginasi
- Response:
```json
{
  "success": true,
  "message": "Logs retrieved successfully",
  "data": {
    "total": 1,
    "limit": 20,
    "items": [
      {
        "id": "log-7b649d883907e59b",
        "timestamp": "2026-09-27T15:45:04Z",
        "actor": "admin-01",
        "action": "REGISTER_DEVICE",
        "method": "POST",
        "path": "/api/v1/devices",
        "client_ip": "127.0.0.1",
        "user_agent": "PostmanRuntime/7.32.3",
        "status_code": 201,
        "duration_ms": 3,
        "request_body": "{\"name\":\"Edge-Device-001\",\"type\":\"gateway\",\"ip_address\":\"192.168.1.10\"}",
        "response_body": "{\"success\":true,\"message\":\"Device created successfully\"...}",
        "error_message": "",
        "status": "SUCCESS"
      }
    ]
  }
}
```

#### Detail Action Log by ID
- **`GET /api/v1/logs/{id}`**

---

### 3. Device Management

#### Registrasi Device Baru
- **`POST /api/v1/devices`**
- Headers:
  - `Content-Type: application/json`
  - `X-API-Key: <MASTER_API_KEY>` atau `Authorization: Bearer <JWT_TOKEN>`
- Body:
```json
{
  "name": "My-ThinkPad-X1",
  "type": "laptop",
  "ip_address": "192.168.1.15",
  "mac_address": "00:1B:44:11:3A:B7",
  "metadata": {
    "os": "Windows 11 / Linux Dual Boot",
    "owner": "Giovanni"
  }
}
```

#### List Semua Device
- **`GET /api/v1/devices`**

#### Detail Device
- **`GET /api/v1/devices/{id}`**

#### Update Device
- **`PUT /api/v1/devices/{id}`**

#### Hapus Device
- **`DELETE /api/v1/devices/{id}`**

---

### 4. Heartbeat & Remote Command

#### Ping Heartbeat Device
- **`POST /api/v1/devices/{id}/heartbeat`**

#### Mengirim Perintah ke Device
- **`POST /api/v1/devices/{id}/commands`**
- Body: `{"payload": "lock_screen"}`

#### Update Hasil Eksekusi Perintah
- **`POST /api/v1/commands/{id}/result`**
- Body:
```json
{
  "status": "success",
  "result": "Laptop screen locked successfully"
}
```

---

### 5. Telemetry Perangkat
- **`POST /api/v1/devices/{id}/telemetry`**
- **`GET /api/v1/devices/{id}/telemetry`**
