# Migrasi Apigateway: Echo → Chi

> **Lingkup:** `microservice-pointofsale-grpc/service/apigateway` (+ harness `tests/` dan `pkg/upload_image`)
> **Router:** `github.com/labstack/echo/v4` → `github.com/go-chi/chi/v5`
> **Status:** ✅ Selesai — build ✅ · vet ✅ · unit test ✅ (10 paket)
> **Pola:** sama dengan migrasi apigateway `monolith-pointofsale-grpc` (lihat `../../monolith-pointofsale-grpc/service/apigateway/MIGRATION_ECHO_TO_CHI.md`); sibling `microservice-ecommerce-grpc` dan `microservice-payment-gateway-grpc` juga sudah chi — repo ini yang terakhir.

---

## 1. Ringkasan

| Item | Jumlah |
|---|---|
| File yang import echo (sebelum) | 17 (1 bootstrap, 13 handler, 3 middleware) |
| Handler domain yang dimigrasi | 12 (`auth`, `cashier`, `category`, `merchant`, `merchant_document`, `order`, `order_item`, `product`, `role`, `transaction`, `user`, **`stats`**) — struktur flat, satu paket |
| Rute terdaftar (terkonversi) | 124 (54 GET · 60 POST · 10 DELETE) — terverifikasi identik dengan baseline |
| Middleware global | 8 (`Recoverer → RequestID → Logger → TraceContext → CORS → Compress(5) → Secure → JWTAuth`) |
| Middleware per-route | — (semua rute dibungkus `apiHandler.Handle`, tanpa chain) |
| Test harness `tests/` yang dikonversi | 11 file (+ `mock_upload.go`) |

Perbedaan dari monolith POS: ada handler `stats` tambahan (5 group: order/product/category/transaction/cashier-stats, 7 rute) yang dikonversi manual tanpa referensi langsung; jumlah rute lebih sedikit (124 vs 169); file microservice dijadikan sumber kebenaran (mis. `transaction.go` microservice tidak punya rute monthly/yearly stats seperti monolith — tidak ditambahkan).

## 2. Karakteristik Khusus POS

1. **Error semantics gaya payment-gateway** — `HTTPErrorHandler` generik (`{"status":"error","message":...,"code":...}`), `shared.HandleApiError` tanpa penanganan `*echo.HTTPError`. Adapter `httpx.Handler`/`apierror.HandleApiError` yang dipakai adalah versi payment-gateway.
2. **Context key `userID`** — `JWTAuth()` chi (pengganti `WebSecurityConfig` echo-jwt) menyimpan claim `sub` ke context key `"userID"`, sama seperti `SuccessHandler` echo-jwt lama.
3. **`TraceContextMiddleware`** — ekstraksi W3C trace context, di-rewrite ke chi tanpa `wrapResponseWriter` (tidak membaca status).
4. **Upload gambar** — `pkg/upload_image.ProcessImageUpload(c echo.Context, file)` → `(w http.ResponseWriter, file)` dengan helper `writeJSONError`; dipakai handler `product` (`c.FormValue`/`c.FormFile` → `r.FormValue`/`r.FormFile`, `r.FormFile` return 3 nilai).
5. **Helper `parseQueryInt`/`parseQueryStringRequired`/`parseQueryIntWithValidation`** — signature `echo.Context` → `*http.Request`.
6. **RateLimiter** — tidak terpasang di chain global (dead code di bootstrap), tetap dikonversi agar konsisten.

## 3. Dependensi

**Ditambah:** `go-chi/chi/v5 v5.3.2`, `go-chi/cors v1.2.2`, `golang-jwt/jwt/v5 v5.2.2`, `swaggo/http-swagger v1.3.4`.
**Tidak lagi di-import langsung:** `labstack/echo/v4`, `echo-jwt/v4`, `echo-swagger` (masih *indirect* via `shared/errors` echo-typed yang dipakai modul `tests/`).

## 4. Struktur Baru

```
service/apigateway/
├── apierror/          # NEW — ApiHandler versi net/http (replaces shared errors.ApiHandler)
├── httpx/             # NEW — helper JSON/Bind/RealIP + request-scoped values
├── apps/client.go     # bootstrap chi (Recoverer → RequestID → Logger → Trace → CORS → Compress → Secure → JWT)
├── handler/           # 14 file flat → chi.Router (12 domain + handler.go + helpers)
└── middlewares/       # auth (JWT), ratemiddleware, trace
pkg/upload_image/      # ProcessImageUpload(w http.ResponseWriter, file)
tests/                 # 11 file harness terkonversi + mock_upload.go
```

## 5. Verifikasi

```bash
cd microservice-pointofsale-grpc/service/apigateway
go build ./... && go vet ./...   # ✅
cd ../../shared && go build ./... # ✅ tidak berubah
cd ../pkg && go build ./...       # ✅ upload_image baru
cd ../tests && go build ./... && go vet ./... && go test ./...  # ✅ 10 paket ok
```

Unit test hijau: `auth` (104s), `cashier` (92s), `category` (91s), `merchant` (104s), `order` (99s), `order_item` (87s), `product` (87s), `role` (80s), `transaction` (31s), `user` (31s).

Harness `tests/` (11 file): `NewHandlerX(s.echo, …)` → `NewHandlerX(s.echo /* chi.Router */, …)`, bypass middleware `c.Set(...)` → `context.WithValue` (auth test kini set key `"userID"` yang benar-benar dibaca handler `/api/auth/me`), `MockImageUpload` ikut signature baru, `apierror.NewApiHandler` menggantikan `errors.NewApiHandler` shared.

## 6. Catatan Perilaku

1. **Respons error identik** — 404/405 JSON eksplisit `{"status":"error",...}` via `r.NotFound`/`r.MethodNotAllowed` (bentuk sama dengan error handler echo lama), error `AppError` dari rute wrapped tetap JSON `ErrorResponse` dengan trace_id dari span OTel.
2. **Sintaks path param** — `:id` → `{id}` (chi tidak mendukung `:param`); `chi.URLParam(r, "id")` tetap pakai nama param yang sama. Rute root `""` → `"/"`.
3. **Swagger** — `echoSwagger.WrapHandler` → `httpSwagger.WrapHandler` di `/swagger/*`; anotasi `@Router` di komentar swagger tidak diubah.
4. **CORS** — whitelist origins/methods/headers dari config echo dipertahankan (`X-API-Key`, bukan `X-Request-ID` seperti monolith), tambah preflight `MaxAge 86400`.
5. **Echo masih indirect dependency** via `shared/errors`.

## 7. Follow-up (Opsional)

- [ ] Smoke test `hurl/` atau full stack docker.
- [ ] Migrasi `shared/errors` penuh ke net/http → echo hilang total dari repo.
- [ ] Ketiga microservice apigateway kini konsisten memakai chi; pertimbangkan menyamakan lokasi package fondasi (`httpx`/`apierror`) dan versi chi lintas repo.
