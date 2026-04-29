# 認証フロー シーケンス図

## 1. ユーザー登録 (POST /api/auth/register)

```mermaid
sequenceDiagram
    actor Client
    participant Handler as AuthHandler
    participant Usecase as AuthUsecase
    participant UserRepo as UserRepository
    participant TokenRepo as VerificationTokenRepository
    participant Email as EmailClient (Resend)
    participant DB as PostgreSQL

    Client->>Handler: POST /api/auth/register<br/>{email, password, name}
    Handler->>Handler: ShouldBindJSON バリデーション<br/>(required, email, min=8)

    alt バリデーション失敗
        Handler-->>Client: 400 Bad Request
    end

    Handler->>Usecase: Register(email, password, name)
    Usecase->>UserRepo: FindByEmail(email)
    UserRepo->>DB: SELECT FROM users WHERE email=?
    DB-->>UserRepo: ErrNoRows

    alt メール重複
        DB-->>UserRepo: User
        UserRepo-->>Usecase: entity.User
        Usecase-->>Handler: ErrAlreadyExists
        Handler-->>Client: 409 Conflict
    end

    UserRepo-->>Usecase: ErrNotFound
    Usecase->>Usecase: bcrypt.GenerateFromPassword(password)
    Usecase->>UserRepo: Create(user)
    UserRepo->>DB: INSERT INTO users
    DB-->>UserRepo: User
    UserRepo-->>Usecase: entity.User

    Usecase->>TokenRepo: DeleteByEmail(email)
    TokenRepo->>DB: DELETE FROM verification_tokens WHERE email=?
    Usecase->>Usecase: rand.Read(32bytes) → hex token
    Usecase->>TokenRepo: Create(verificationToken, expires=+1h)
    TokenRepo->>DB: INSERT INTO verification_tokens
    Usecase->>Email: SendVerificationEmail(email, token, appBaseURL)
    Email-->>Usecase: nil

    Usecase-->>Handler: nil
    Handler-->>Client: 201 Created<br/>{message: "registration successful. please check your email."}
```

---

## 2. ログイン (POST /api/auth/login)

```mermaid
sequenceDiagram
    actor Client
    participant Handler as AuthHandler
    participant Usecase as AuthUsecase
    participant UserRepo as UserRepository
    participant JWT as JWTManager
    participant DB as PostgreSQL

    Client->>Handler: POST /api/auth/login<br/>{email, password}
    Handler->>Handler: ShouldBindJSON バリデーション

    alt バリデーション失敗
        Handler-->>Client: 400 Bad Request
    end

    Handler->>Usecase: Login(email, password)
    Usecase->>UserRepo: FindByEmail(email)
    UserRepo->>DB: SELECT FROM users WHERE email=?

    alt ユーザー未存在
        DB-->>UserRepo: ErrNoRows
        UserRepo-->>Usecase: ErrNotFound
        Usecase-->>Handler: ErrUnauthorized
        Handler-->>Client: 401 Unauthorized
    end

    DB-->>UserRepo: User
    UserRepo-->>Usecase: entity.User

    alt PasswordHash が nil (SSOユーザー)
        Usecase-->>Handler: ErrUnauthorized
        Handler-->>Client: 401 Unauthorized
    end

    Usecase->>Usecase: bcrypt.CompareHashAndPassword

    alt パスワード不一致
        Usecase-->>Handler: ErrUnauthorized
        Handler-->>Client: 401 Unauthorized
    end

    alt メール未確認 (EmailVerifiedAt == nil)
        Usecase-->>Handler: ErrUnauthorized
        Handler-->>Client: 401 Unauthorized
    end

    Usecase->>JWT: GenerateAccessToken(userID) → 15分
    Usecase->>JWT: GenerateRefreshToken(userID) → 7日
    JWT-->>Usecase: TokenPair

    Usecase-->>Handler: TokenPair
    Handler-->>Client: 200 OK<br/>{access_token, refresh_token}
```

---

## 3. メール確認 (GET /api/auth/verify-email?token=xxx)

```mermaid
sequenceDiagram
    actor Client
    participant Handler as AuthHandler
    participant Usecase as AuthUsecase
    participant TokenRepo as VerificationTokenRepository
    participant UserRepo as UserRepository
    participant DB as PostgreSQL

    Client->>Handler: GET /api/auth/verify-email?token=xxx
    Handler->>Handler: token クエリパラメータチェック

    alt token 空
        Handler-->>Client: 400 Bad Request
    end

    Handler->>Usecase: VerifyEmail(token)
    Usecase->>TokenRepo: FindByToken(token)
    TokenRepo->>DB: SELECT FROM verification_tokens WHERE token=?

    alt トークン未存在
        DB-->>TokenRepo: ErrNoRows
        TokenRepo-->>Usecase: ErrNotFound
        Usecase-->>Handler: ErrInvalidToken
        Handler-->>Client: 400 Bad Request
    end

    DB-->>TokenRepo: VerificationToken
    TokenRepo-->>Usecase: entity.VerificationToken

    alt トークン期限切れ (ExpiresAt < now)
        Usecase-->>Handler: ErrInvalidToken
        Handler-->>Client: 400 Bad Request
    end

    Usecase->>UserRepo: FindByEmail(token.Email)
    UserRepo->>DB: SELECT FROM users WHERE email=?
    DB-->>UserRepo: User
    UserRepo-->>Usecase: entity.User

    Usecase->>UserRepo: UpdateEmailVerified(userID, now)
    UserRepo->>DB: UPDATE users SET email_verified_at=now WHERE id=?
    DB-->>UserRepo: User

    Usecase->>TokenRepo: DeleteByEmail(email)
    TokenRepo->>DB: DELETE FROM verification_tokens WHERE email=?

    Usecase-->>Handler: nil
    Handler-->>Client: 200 OK<br/>{message: "email verified successfully"}
```

---

## 4. トークンリフレッシュ (POST /api/auth/refresh)

```mermaid
sequenceDiagram
    actor Client
    participant Handler as AuthHandler
    participant Usecase as AuthUsecase
    participant JWT as JWTManager
    participant UserRepo as UserRepository
    participant DB as PostgreSQL

    Client->>Handler: POST /api/auth/refresh<br/>{refresh_token}
    Handler->>Handler: ShouldBindJSON バリデーション

    alt バリデーション失敗
        Handler-->>Client: 400 Bad Request
    end

    Handler->>Usecase: RefreshToken(refreshToken)
    Usecase->>JWT: ValidateRefreshToken(refreshToken)

    alt トークン無効・期限切れ
        JWT-->>Usecase: error
        Usecase-->>Handler: ErrInvalidToken
        Handler-->>Client: 401 Unauthorized
    end

    JWT-->>Usecase: Claims{UserID}

    Usecase->>UserRepo: FindByID(claims.UserID)
    UserRepo->>DB: SELECT FROM users WHERE id=?

    alt ユーザー未存在 (退会済み等)
        DB-->>UserRepo: ErrNoRows
        UserRepo-->>Usecase: ErrNotFound
        Usecase-->>Handler: ErrUnauthorized
        Handler-->>Client: 401 Unauthorized
    end

    DB-->>UserRepo: User
    UserRepo-->>Usecase: entity.User

    Usecase->>JWT: GenerateAccessToken(userID) → 15分
    Usecase->>JWT: GenerateRefreshToken(userID) → 7日
    JWT-->>Usecase: TokenPair

    Usecase-->>Handler: TokenPair
    Handler-->>Client: 200 OK<br/>{access_token, refresh_token}
```

---

## 5. 認証済みAPIアクセス (JWTミドルウェア)

```mermaid
sequenceDiagram
    actor Client
    participant Middleware as JWTAuth Middleware
    participant JWT as JWTManager
    participant Handler as Protected Handler

    Client->>Middleware: GET /api/protected<br/>Authorization: Bearer <access_token>

    alt Authorization ヘッダなし
        Middleware-->>Client: 401 Unauthorized
    end

    alt Bearer フォーマット不正
        Middleware-->>Client: 401 Unauthorized
    end

    Middleware->>JWT: ValidateAccessToken(token)

    alt トークン無効・期限切れ
        JWT-->>Middleware: error
        Middleware-->>Client: 401 Unauthorized
    end

    JWT-->>Middleware: Claims{UserID}
    Middleware->>Middleware: c.Set("userID", claims.UserID)
    Middleware->>Handler: c.Next()
    Handler-->>Client: 200 OK (レスポンス)
```
