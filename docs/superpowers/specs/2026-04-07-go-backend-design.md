# Go Backend Design Spec

**Date:** 2026-04-07
**Project:** Fishing-api（FishingConditionsApp バックエンド分離）
**Status:** Approved

---

## 概要

FishingConditionsAppのバックエンドをNext.js API Routesから独立したGoサービスへ**ビッグバン移行**する。既存のフロントエンド（Next.js）はそのままに、APIエンドポイントをGoで再実装し、k8s Ingressでルーティングを切り替える。

---

## 技術スタック

| 項目 | 技術 |
|------|------|
| 言語 | Go 1.24+ |
| フレームワーク | Gin |
| DB アクセス | sqlc + PostgreSQL |
| キャッシュ | Redis |
| 認証 | JWT（golang-jwt/jwt）+ Google OAuth（golang.org/x/oauth2） |
| コンテナ | Docker（linux/arm64） |
| オーケストレーション | k3s（Raspberry Pi） |
| 外部公開 | Cloudflare Tunnel |
| CI/CD | GitHub Actions → GHCR → Cloudflare Tunnel → kubectl |

---

## アーキテクチャ：クリーンアーキテクチャ

依存関係のルール：**内側の層は外側の層を知らない**。

```
interface → usecase → domain ← infrastructure
```

### 依存関係図

```
[Cloudflare Edge]
      ↓ HTTPS
[cloudflared] → [Traefik Ingress]
                    ├── /api/* → [Gin Handler]
                    │               ↓
                    │           [Middleware: JWT/CORS/RateLimit]
                    │               ↓
                    │           [Usecase] → [Domain Interface]
                    │                            ↑
                    │               [sqlc/Postgres] [Redis]
                    │               [Weather/Tide/Maps API]
                    │               [Email: Resend]
                    │
                    └── /*    → [Next.js Frontend]
```

---

## ディレクトリ構造

```
fishing-api/
├── cmd/
│   └── server/
│       └── main.go              # エントリーポイント・DI組み立て
├── internal/
│   ├── domain/                  # 最内層：変更頻度が最も低い
│   │   ├── entity/              # User, Location, Port, FishingScore等
│   │   └── repository/          # DBアクセスインターフェース（抽象）
│   ├── usecase/                 # ビジネスロジック層
│   │   ├── auth/                # 登録・ログイン・トークン管理・メール認証
│   │   ├── weather/             # 天気データ取得・Redisキャッシュ
│   │   ├── tide/                # 潮汐データ取得・Redisキャッシュ
│   │   ├── location/            # 場所検索（Google Maps）
│   │   ├── favorite/            # お気に入りCRUD
│   │   └── score/               # 釣りやすさスコア算出
│   ├── infrastructure/          # 具体的な実装（差し替え可能）
│   │   ├── db/                  # sqlc生成コード + PostgreSQL接続
│   │   ├── cache/               # Redisクライアント実装
│   │   ├── external/            # OpenWeatherMap・tide736.net・Google Maps
│   │   └── email/               # Resend APIメール送信
│   └── interface/               # HTTPハンドラー層
│       ├── handler/             # Ginルートハンドラー
│       ├── middleware/          # JWT認証・ロギング・CORS・RateLimit
│       └── router/              # ルーティング定義
├── pkg/                         # 外部公開可能なユーティリティ
│   ├── jwt/                     # JWT生成・検証
│   └── validator/               # 座標・UUID等バリデーション
├── db/
│   ├── schema.sql               # DBリポジトリからCI/CDで自動同期
│   ├── queries/                 # sqlc用SQLクエリ（手動管理）
│   └── generated/               # sqlcが自動生成するGoコード
├── k8s/
│   ├── namespace.yaml
│   ├── fishing-api/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── ingress.yaml         # /api/* → fishing-api
│   ├── frontend/
│   │   ├── deployment.yaml
│   │   ├── service.yaml
│   │   └── ingress.yaml         # /* → frontend
│   ├── cloudflared/
│   │   └── deployment.yaml
│   └── config/
│       ├── postgres-secret.yaml
│       └── redis-secret.yaml
├── docs/
│   ├── architecture.md
│   ├── api.md
│   ├── auth.md
│   └── development.md
├── .github/
│   └── workflows/
│       ├── ci.yml               # Lint・Test・Build
│       ├── deploy.yml           # GHCR push → kubectl apply
│       └── sync-schema.yml      # DBリポジトリからschema.sql同期
├── sqlc.yaml
└── config/                      # 環境変数・設定管理
```

---

## ドメイン層

### エンティティ（`internal/domain/entity/`）

既存PrismaスキーマをGoエンティティに1:1対応させる。

```go
// User: 認証・プロフィール
type User struct {
    ID              uuid.UUID
    Email           string
    PasswordHash    string
    Name            string
    AvatarURL       *string
    IsSSO           bool
    EmailVerifiedAt *time.Time
    CreatedAt       time.Time
    UpdatedAt       time.Time
}

// Location: 釣り場マスタ
type Location struct {
    ID         uuid.UUID
    Name       string
    Latitude   float64
    Longitude  float64
    Region     string
    Prefecture string
    Type       LocationType  // SHORE/SURF/PORT/RIVER/LAKE/OFFSHORE/OTHER
    PortID     *uuid.UUID
}

// FishingScore: 計算結果（DB永続化なし）
type FishingScore struct {
    Total        int        // 0-100
    Rank         ScoreRank  // excellent/good/fair/poor/bad
    TideScore    int        // 最大40点
    WeatherScore int        // 最大35点
    TimeScore    int        // 最大25点
    Explanation  string
}

// WeatherData: 天気データ（外部API結果・DB永続化なし）
type WeatherData struct {
    Temperature float64
    FeelsLike   float64
    WindSpeed   float64
    WindDeg     int
    Pressure    float64
    Humidity    int
    Description string
    DateTime    time.Time
}

// TideData: 潮汐データ（外部API結果・DB永続化なし）
type TideData struct {
    PortCode    string
    Date        string
    HighTides   []TideEvent
    LowTides    []TideEvent
    TideType    string  // 大潮/中潮/小潮/長潮/若潮
}

type TideEvent struct {
    Time   time.Time
    Height float64
}

// VerificationToken: メール検証トークン
type VerificationToken struct {
    Token     string
    Email     string
    ExpiresAt time.Time
    CreatedAt time.Time
}
```

### リポジトリインターフェース（`internal/domain/repository/`）

```go
type UserRepository interface {
    FindByEmail(ctx context.Context, email string) (*entity.User, error)
    FindByID(ctx context.Context, id uuid.UUID) (*entity.User, error)
    Create(ctx context.Context, user *entity.User) error
    UpdateEmailVerified(ctx context.Context, id uuid.UUID, verifiedAt time.Time) error
}

type FavoriteRepository interface {
    FindByUserID(ctx context.Context, userID uuid.UUID) ([]*entity.Location, error)
    Add(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error
    Delete(ctx context.Context, userID uuid.UUID, locationID uuid.UUID) error
}

type VerificationTokenRepository interface {
    Create(ctx context.Context, token *entity.VerificationToken) error
    FindByToken(ctx context.Context, token string) (*entity.VerificationToken, error)
    DeleteByEmail(ctx context.Context, email string) error
}
```

---

## ユースケース層

### 認証（`internal/usecase/auth/`）

```go
type AuthUsecase interface {
    Register(ctx context.Context, email, password, name string) error
    Login(ctx context.Context, email, password string) (*TokenPair, error)
    RefreshToken(ctx context.Context, refreshToken string) (*TokenPair, error)
    VerifyEmail(ctx context.Context, token string) error
    LinkGoogleAccount(ctx context.Context, googleIDToken string, userID uuid.UUID) error
}

type TokenPair struct {
    AccessToken  string  // 有効期限: 15分
    RefreshToken string  // 有効期限: 7日
}
```

### 天気・潮汐

```go
type WeatherUsecase interface {
    GetCurrent(ctx context.Context, lat, lon float64) (*entity.WeatherData, error)
    GetForecast(ctx context.Context, lat, lon float64) ([]*entity.WeatherData, error)
}

type TideUsecase interface {
    GetTideData(ctx context.Context, prefCode, portCode, date string) (*entity.TideData, error)
}
```

### スコア算出

既存TypeScript実装のロジックをそのままGoに移植する。

```
Total(0-100) = 潮汐スコア(40) + 天気スコア(35) + 時間帯スコア(25)
```

### キャッシュ戦略

| データ | Redisキーパターン | TTL |
|--------|------------------|-----|
| 天気（現在） | `weather:{lat}:{lon}` | 30分 |
| 天気（予報） | `weather:forecast:{lat}:{lon}` | 30分 |
| 潮汐 | `tide:{portCode}:{date}` | 6時間 |
| 場所検索 | `location:{query}` | 1日 |

---

## インターフェース層

### APIエンドポイント

```
POST   /api/auth/register
POST   /api/auth/login
POST   /api/auth/refresh
GET    /api/auth/verify-email?token=xxx
POST   /api/auth/google

GET    /api/weather?lat=xx&lon=xx&type=current|forecast
GET    /api/conditions/tide?prefectureCode=xx&portCode=xx&date=xx

GET    /api/locations/search?q=xx&limit=xx
GET    /api/favorites
POST   /api/favorites
DELETE /api/favorites/:id
```

### ミドルウェア

| ミドルウェア | 役割 |
|-------------|------|
| JWTAuthMiddleware | AuthorizationヘッダーからJWT検証 → ctxにuserID注入 |
| LoggingMiddleware | リクエスト/レスポンスのログ出力 |
| CORSMiddleware | Next.jsフロントエンドからのアクセス許可 |
| RateLimitMiddleware | 外部API過剰呼び出し防止 |

### エラーレスポンス形式

既存フロントエンドとの互換性を保つため、同一形式を維持する。

```json
{
  "error": "エラーメッセージ",
  "code": "ERROR_CODE",
  "status": 400
}
```

---

## インフラストラクチャ層

### sqlc設定

```yaml
# sqlc.yaml
version: "2"
sql:
  - engine: "postgresql"
    queries: "db/queries/"
    schema: "db/schema.sql"
    gen:
      go:
        package: "sqlcgen"
        out: "db/generated/"
        emit_interface: true
```

### 外部APIクライアント

タイムアウト10秒・最大3回リトライを標準設定とする。

```go
type WeatherClient interface {
    FetchCurrent(ctx context.Context, lat, lon float64) (*WeatherResponse, error)
    FetchForecast(ctx context.Context, lat, lon float64) (*ForecastResponse, error)
}

type TideClient interface {
    FetchTideData(ctx context.Context, prefCode, portCode, date string) (*TideResponse, error)
}

type MapsClient interface {
    SearchLocations(ctx context.Context, query string, limit int) ([]*LocationResult, error)
}
```

---

## デプロイメント構成

### インフラ全体図

```
[Internet]
    ↓ HTTPS
[Cloudflare Edge] ←──── cloudflared tunnel ────┐
                                                │
[Raspberry Pi k3s] ──────────────────────────────
    ├── cloudflared Pod     # Tunnel エージェント
    ├── Traefik Ingress      # k3sデフォルトIngress
    ├── fishing-api Pod      # Goバックエンド（ARM64）
    ├── frontend Pod         # Next.js（ARM64）
    ├── postgres Pod
    └── redis Pod
```

### Cloudflare Tunnelの役割

| 用途 | 説明 |
|------|------|
| アプリ公開 | `fishing.kazuma-lab.com` → k3s Traefik |
| CI/CDアクセス | GitHub Actions → k3s API Server（Cloudflare Access経由） |
| SSL/TLS | Cloudflareが自動管理 |

### CI/CDパイプライン

```
コードpush (main)
    ↓
① Lint / Test / go build
    ↓
② Docker buildx（linux/arm64）
    ↓
③ GHCR（ghcr.io/kazumadev619-dev/fishing-api）にpush
    ↓
④ cloudflared access でk3s APIサーバーにトンネル接続
    ↓
⑤ kubectl set image でローリングデプロイ
```

### DB schema.sql 自動同期フロー

```
DBリポジトリ push
    ↓
GitHub Actions（DBリポジトリ側）が repository_dispatch 送信
    ↓
fishing-api の sync-schema.yml が起動
    ↓
schema.sql を更新 → sqlc generate → PR自動作成
```

---

## テスト戦略

| 層 | 手法 | ツール |
|----|------|--------|
| usecase | モックリポジトリでユニットテスト | testify/mock |
| infrastructure/db | Docker上の実PostgreSQLで統合テスト | testcontainers-go |
| infrastructure/external | HTTPモックサーバー | net/http/httptest |
| interface/handler | ルーター全体の統合テスト | httptest + Gin |

---

## 移行戦略

**ビッグバン移行**：Next.js API Routesを全廃し、GoバックエンドですべてのAPIを再実装する。

### 移行手順

1. Goバックエンドをローカル開発環境で完成させ、全APIを実装
2. k3s上にGoバックエンドをデプロイ（Next.jsと並走）
3. k8s IngressのルールをGoバックエンドに切り替え
4. Next.jsのAPI Routesを削除

### フロントエンドへの影響

k8s Ingressで `/api/*` をGoバックエンドに転送するため、フロントエンドのコード変更は不要。

---

## 確定事項（旧・未決定事項）

- DBリポジトリ: `Fishing-database`
- ハードウェア: Raspberry Pi 5 / Ubuntu Server 24.04.4 64bit（ARM64）
- ドメイン: `kazuma-lab.com`（アプリ: `fishing.kazuma-lab.com`、k3s API: `k3s-api.kazuma-lab.com`）
- Redis・PostgreSQLのデプロイ方式: Raspberry Pi上のDocker Compose（k3s外）で常時運用。k8sベストプラクティス（PodはEphemeral）に従い、DBはStatefulSet管理しない。外部マネージドサービスも使用しない。systemdでPi再起動時の自動復旧を設定する。
