# DAKAI Backend Refactoring Plan

## 概要

本ドキュメントは、Go実装のDAKAIバックエンドをJava (Spring Boot + MyBatis) で
リファクタリングするにあたり、現状の問題点と改善方針を定義する。

---

## 1. なぜリファクタリングするのか

### 1.1 学習目的

- Java/Spring Boot エコシステムの習得
- 認証基盤（JWT）の自前実装による深い理解
- TDDによる開発プロセスの体得
- スレッドセーフ・並行処理の実践
- MySQLパフォーマンスチューニングの学習
- インデックス設計の実践

### 1.2 担当分担

| 領域 | 担当 | 備考 |
|------|------|------|
| **OpenAPI整理** | AI | 仕様の整理・再設計 |
| **フロントエンド** | AI | AI駆動で構築 |
| **バックエンド (Java)** | 自分 | AIは補助的に使用、学習メイン |

### 1.2 現実装の技術的負債

現Go実装には以下の問題があり、学習教材としてこれらを改善する。

---

## 2. 現プロジェクトの全体像

### 2.1 技術スタック（現状）

| 項目 | 現状（Go） |
|------|-----------|
| 言語 | Go 1.23.5 |
| Webフレームワーク | Gin |
| DB | MySQL + sqlx |
| 認証 | Firebase Auth |
| ストレージ | AWS S3 |
| AI連携 | OpenAI API (gpt-image-1) |
| アーキテクチャ | レイヤードアーキテクチャ（Handler → Service → Repository/Adapter） |

### 2.2 アーキテクチャ図（現状）

```
┌─────────────────────────────────────────────────────────────────────────┐
│                            Handler Layer                                │
│  ┌──────────────────┐    ┌───────────────────┐                         │
│  │   UserHandler    │    │   RecordHandler   │                         │
│  │  - SignUp        │    │  - GetRecords     │                         │
│  │  - Login         │    │  - DeleteRecord   │                         │
│  │  - GetProfile    │    │  - CreateLike     │                         │
│  │  - Follow/Unfollow│   │  - CheckLike      │                         │
│  │  - GetFollowing  │    │  - DeleteLike     │                         │
│  │  - GetFollowed   │    │  - GenerateIllust │                         │
│  └────────┬─────────┘    └─────────┬─────────┘                         │
└───────────┼─────────────────────────┼───────────────────────────────────┘
            │                         │
            ▼                         ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                            Service Layer                                │
│  ┌──────────────────┐    ┌───────────────────┐                         │
│  │   UserService    │    │   RecordService   │                         │
│  │  (users.go)      │    │   (records.go)    │                         │
│  └────────┬─────────┘    └─────────┬─────────┘                         │
│           │                        │                                    │
│           │    ┌───────────────────┴────────────────┐                  │
│           │    │         ports.go (Interface)       │                  │
│           │    │  - UserQueryRepo                   │                  │
│           │    │  - UserCommandRepo                 │                  │
│           │    │  - RecordQueryRepo                 │                  │
│           │    │  - RecordCommandRepo               │                  │
│           │    │  - ProfileRepo                     │                  │
│           │    │  - AuthClient                      │                  │
│           │    │  - AwsClient                       │                  │
│           │    │  - GptClient                       │                  │
│           │    └────────────────────────────────────┘                  │
└───────────┼─────────────────────────┼───────────────────────────────────┘
            │                         │
            ▼                         ▼
┌───────────────────────────┐  ┌──────────────────────────────────────────┐
│      Repository (dbase)   │  │           Adapter                        │
│  ┌─────────────────────┐  │  │  ┌────────────────┐                      │
│  │ user_query_repo     │  │  │  │ firebase_auth  │ → Firebase Auth      │
│  │ user_command_repo   │  │  │  ├────────────────┤                      │
│  │ record_query_repo   │  │  │  │ aws_client     │ → AWS S3             │
│  │ record_command_repo │  │  │  ├────────────────┤                      │
│  │ profile_repo        │  │  │  │ gpt_client     │ → OpenAI API         │
│  └──────────┬──────────┘  │  │  └────────────────┘                      │
└─────────────┼─────────────┘  └──────────────────────────────────────────┘
              │
              ▼
┌─────────────────────────────┐
│          MySQL              │
│  ┌───────────────────────┐  │
│  │ characters            │  │
│  │ users                 │  │
│  │ profiles              │  │
│  │ records               │  │
│  │ user_likes            │  │
│  │ follows               │  │
│  └───────────────────────┘  │
└─────────────────────────────┘
```

### 2.3 DBスキーマ（ER図）

```
┌─────────────────┐
│   characters    │
├─────────────────┤
│ id (PK)         │
│ name            │
│ image_url       │
│ level           │
│ current_point   │
│ limit_point     │
└────────┬────────┘
         │ 1
         │
         │ *
┌────────┴────────┐       ┌─────────────────┐
│     users       │       │    profiles     │
├─────────────────┤       ├─────────────────┤
│ id (PK)         │──1:1──│ id (PK)         │
│ character_id(FK)│       │ user_id (FK)    │
│ firebase_uid    │       │ profile_image   │
│ name            │       └─────────────────┘
└────────┬────────┘
         │
    ┌────┴────┬──────────────────┐
    │ 1       │ *                │ *
    │         │                  │
    │    ┌────┴────┐       ┌─────┴─────┐
    │    │ records │       │  follows  │
    │    ├─────────┤       ├───────────┤
    │    │ id (PK) │       │follower_id│
    │    │ user_id │       │followed_id│
    │    │object_key│      └───────────┘
    │    │clean_time│
    │    │clean_date│
    │    │ comment  │
    │    └────┬─────┘
    │         │ 1
    │         │
    │    ┌────┴──────┐
    │    │user_likes │
    │    ├───────────┤
    └────│ user_id   │
         │ record_id │
         └───────────┘
```

### 2.4 機能一覧

| 機能 | Handler | Service | Repository | 外部連携 |
|------|---------|---------|------------|----------|
| ユーザー登録 | `SignUpUserById` | `SignUpUser` | `CreateUserById` | Firebase Auth |
| ログイン | `LoginUser` | `LoginUser` | `FindByToken` | Firebase Auth |
| プロフィール取得 | `GetProfileById` | `GetProfile` | `GetProfileById` | - |
| フォロー | `FollowUser` | `FollowUser` | `FollowUserById` | Firebase Auth |
| フォロー解除 | `DeleteFollowUser` | `DeleteFollowUser` | `DeleteFollowUserById` | Firebase Auth |
| フォロー中一覧 | `GetFollowing` | `GetFollowingById` | `GetFollowingById` | - |
| フォロワー一覧 | `GetFollowed` | `GetFollowedById` | `GetFollowedById` | - |
| フォローチェック | `CheckFollow` | `CheckFollowById` | `CheckFollowById` | Firebase Auth |
| 記録一覧（全体） | `GetRecords` | `GetRecords` | `GetRecords` | AWS S3 (presigned) |
| 記録一覧（個人） | `GetRecordsById` | `GetRecordsById` | `GetRecordsById` | AWS S3 (presigned) |
| 記録削除 | `DeleteRecord` | `DeleteRecordById` | `DeleteRecordById` | Firebase Auth |
| いいね作成 | `CreateLike` | `CreateLike` | `CreateLike` | Firebase Auth |
| いいねチェック | `CheckLike` | `CheckLikeById` | `CheckLike` | Firebase Auth |
| いいね削除 | `DeleteLike` | `DeleteLikeById` | `DeleteLike` | Firebase Auth |
| 画像アップロード＆記録作成 | `GenerateIllustration` | `UploadIllustration` + `CreateRecord` | `CreateRecordById` | OpenAI + AWS S3 |

---

## 3. 現状の問題点

### 3.1 認証・認可

| 問題 | 現状 | 影響 |
|------|------|------|
| Firebase依存 | 認証を全てFirebaseに委譲 | 認証の仕組みを理解できない |
| トークン検証の重複 | 各Handlerで `strings.HasPrefix(authz, "Bearer ")` を繰り返し | DRY原則違反、保守性低下 |
| 認可の欠如 | 他ユーザーのリソース操作を防ぐロジックが不十分 | セキュリティリスク |

**該当コード例** (`handler/users.go`):
```go
// 全Handlerで同じコードが繰り返されている
authz := ctx.GetHeader("Authorization")
if !strings.HasPrefix(authz, "Bearer ") {
    ctx.JSON(http.StatusUnauthorized, gin.H{"error": "missing bearer token"})
    return
}
token := strings.TrimPrefix(authz, "Bearer ")
```

### 3.2 例外処理

| 問題 | 現状 | 影響 |
|------|------|------|
| 例外階層が貧弱 | `ErrVerifyUser` のみ定義 | エラー原因の特定が困難 |
| エラーメッセージの不統一 | `"internal server error"` と `"server internal error"` 混在 | クライアント側のハンドリング困難 |
| ログとレスポンスの乖離 | `log.Println` で詳細出力、クライアントには汎用メッセージ | デバッグ時の追跡困難 |

**該当コード** (`apperrs/errors.go`):
```go
package apperrs

import "errors"

var ErrVerifyUser = errors.New("failed to verify user")
// これ以外のエラー定義がない
```

**Handlerでの処理**:
```go
switch {
case errors.Is(err, apperrs.ErrVerifyUser):
    ctx.AbortWithStatusJSON(http.StatusUnauthorized, ...)
default:
    // 全ての未知エラーが同じ扱い
    ctx.AbortWithStatusJSON(http.StatusBadRequest, gin.H{"error": "internal server error"})
}
```

### 3.3 テスト

| 問題 | 現状 | 影響 |
|------|------|------|
| カバレッジ不足 | Service層の一部のみ | リグレッションリスク |
| 統合テストなし | DBを含むE2Eテストがない | 本番環境での不具合発見遅延 |
| TDDでない | 実装後にテスト追加 | 設計がテスタビリティを考慮していない |

### 3.4 並行処理・スレッドセーフ

| 問題 | 現状 | 影響 |
|------|------|------|
| いいねカウントの競合 | `likes_count` の更新に排他制御なし | カウント不整合 |
| 画像アップロードの同期処理 | S3アップロード完了まで待機 | レスポンス遅延 |

### 3.5 OpenAPI仕様

| 問題 | 詳細 |
|------|------|
| securitySchemes未定義 | `bearerAuth` 参照あるが定義なし |
| パラメータ定義漏れ | path parameter の `parameters` 未定義多数 |
| スキーマ参照の不統一 | `$ref` と `example` 直書きが混在 |
| 命名の揺れ | `Exercise` / `Record` / `RawRecord` |
| RESTful設計違反 | `DELETE /users/unfollows` はリソース指向でない |
| 未使用スキーマ | `Team`, `UserCharacter` は未実装 |

### 3.6 データベース設計

| 問題 | 現状 | 影響 |
|------|------|------|
| インデックス未定義 | 検索条件カラムにINDEXなし | クエリ性能劣化 |
| N+1問題の可能性 | JOINせず個別クエリ | 大量リクエスト時の性能劣化 |
| バージョン管理なし | 楽観的ロック用カラムなし | 更新競合検知不可 |

**現状のテーブル定義例**:
```sql
CREATE TABLE records (
    `id` INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    `user_id` INT NOT NULL,
    -- user_id にINDEXがない
    -- version カラムがない（楽観的ロック不可）
);
```

---

## 4. リファクタリング方針

### 4.1 技術スタック

| 領域 | 採用技術 |
|------|----------|
| 言語 | Java 21 |
| フレームワーク | Spring Boot 3.x |
| DB接続 | MyBatis |
| DB | MySQL 8.x |
| 認証 | 自前JWT実装 |
| テスト | JUnit 5 + Mockito + Testcontainers |
| ビルド | Gradle |
| コンテナ | Docker Compose |

### 4.2 アーキテクチャ

```
┌─────────────────────────────────────────────────────────┐
│                    Controller Layer                     │
│  - リクエスト/レスポンスのマッピング                    │
│  - バリデーション (@Valid)                              │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                    Service Layer                        │
│  - ビジネスロジック                                     │
│  - トランザクション管理 (@Transactional)                │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                    Mapper Layer (MyBatis)               │
│  - SQL定義 (XML or アノテーション)                      │
│  - パラメータバインド                                   │
│  - 結果マッピング                                       │
└─────────────────────┬───────────────────────────────────┘
                      │
┌─────────────────────▼───────────────────────────────────┐
│                    Infrastructure                       │
│  - JWT Provider                                         │
│  - S3 Client                                            │
│  - OpenAI Client                                        │
└─────────────────────────────────────────────────────────┘
```

### 4.3 ディレクトリ構造（Java版）

```
backend-java/
├── build.gradle
└── src/
    ├── main/
    │   ├── java/com/dakai/
    │   │   ├── DakaiApplication.java
    │   │   ├── domain/
    │   │   │   ├── User.java
    │   │   │   ├── Record.java
    │   │   │   └── Profile.java
    │   │   ├── dto/
    │   │   │   ├── request/
    │   │   │   │   ├── SignupRequest.java
    │   │   │   │   └── RecordCreateRequest.java
    │   │   │   └── response/
    │   │   │       ├── UserResponse.java
    │   │   │       └── RecordResponse.java
    │   │   ├── mapper/
    │   │   │   ├── UserMapper.java
    │   │   │   ├── RecordMapper.java
    │   │   │   ├── FollowMapper.java
    │   │   │   └── LikeMapper.java
    │   │   ├── service/
    │   │   │   ├── UserService.java
    │   │   │   ├── RecordService.java
    │   │   │   ├── FollowService.java
    │   │   │   └── LikeService.java
    │   │   ├── controller/
    │   │   │   ├── AuthController.java
    │   │   │   ├── UserController.java
    │   │   │   ├── RecordController.java
    │   │   │   ├── FollowController.java
    │   │   │   └── LikeController.java
    │   │   ├── security/
    │   │   │   ├── JwtProvider.java
    │   │   │   ├── JwtAuthenticationFilter.java
    │   │   │   └── SecurityConfig.java
    │   │   ├── infrastructure/
    │   │   │   ├── S3Client.java
    │   │   │   └── OpenAiClient.java
    │   │   ├── exception/
    │   │   │   ├── AppException.java
    │   │   │   ├── AuthenticationException.java
    │   │   │   ├── ResourceNotFoundException.java
    │   │   │   └── GlobalExceptionHandler.java
    │   │   └── config/
    │   │       ├── AwsConfig.java
    │   │       └── OpenAiConfig.java
    │   └── resources/
    │       ├── application.yml
    │       ├── mapper/
    │       │   ├── UserMapper.xml
    │       │   └── RecordMapper.xml
    │       └── db/migration/
    └── test/
        └── java/com/dakai/
            ├── service/
            ├── controller/
            └── mapper/
```

### 4.4 例外処理設計

```
AppException (RuntimeException)
├── AuthenticationException (401)
│   ├── InvalidTokenException
│   ├── TokenExpiredException
│   └── InvalidCredentialsException
├── AuthorizationException (403)
│   └── AccessDeniedException
├── BusinessException (400)
│   ├── ResourceNotFoundException (404)
│   ├── DuplicateResourceException (409)
│   ├── InvalidRequestException
│   └── OptimisticLockException (409)
└── ExternalServiceException (502)
    ├── StorageException
    └── AiServiceException
```

### 4.5 認証フロー (JWT)

```
[Signup]
Client → POST /auth/signup → UserService.signUp()
       → PasswordEncoder.encode()
       → UserMapper.insert()
       → JwtProvider.generateTokens()
       ← { accessToken, refreshToken }

[Login]
Client → POST /auth/login → UserService.login()
       → UserMapper.findByEmail()
       → PasswordEncoder.matches()
       → JwtProvider.generateTokens()
       ← { accessToken, refreshToken }

[Authenticated Request]
Client → GET /records (Authorization: Bearer <token>)
       → JwtAuthenticationFilter.doFilterInternal()
       → JwtProvider.validateToken()
       → SecurityContextHolder.setAuthentication()
       → Controller → Service → Mapper
       ← Response
```

### 4.6 API設計（整理後）

```yaml
paths:
  # ===== 認証 =====
  /auth/signup:        POST   # ユーザー登録（JWT発行）
  /auth/login:         POST   # ログイン（JWT発行）
  /auth/refresh:       POST   # トークンリフレッシュ

  # ===== ユーザー =====
  /users/{userId}/profile:     GET    # プロフィール取得

  # ===== フォロー =====
  /users/{userId}/following:   GET    # フォロー中一覧
  /users/{userId}/followers:   GET    # フォロワー一覧
  /follows:                    POST   # フォローする
  /follows/{followedId}:       GET    # フォローチェック
  /follows/{followedId}:       DELETE # フォロー解除

  # ===== 記録 =====
  /records:                    GET    # 全体記録一覧
  /records:                    POST   # 記録作成（画像アップロード含む）
  /users/{userId}/records:     GET    # ユーザーの記録一覧
  /records/{recordId}:         DELETE # 記録削除

  # ===== いいね =====
  /records/{recordId}/likes:   GET    # いいねチェック
  /records/{recordId}/likes:   POST   # いいねする
  /records/{recordId}/likes:   DELETE # いいね解除
```

### 4.7 並行処理・スレッドセーフ対策

| 問題 | 対策 |
|------|------|
| いいねカウント競合 | `UPDATE ... SET count = count + 1` + 楽観的ロック |
| 同時フォロー登録 | DB UNIQUE制約 + `DuplicateKeyException` ハンドリング |
| SecurityContext | `MODE_INHERITABLETHREADLOCAL` 設定 |
| 画像アップロード | `@Async` + `CompletableFuture` で非同期化検討 |

---

## 5. MyBatis Mapper について

### 5.1 Mapperとは

MyBatisにおけるMapperは、**SQLとJavaオブジェクトの橋渡し**をする役割。
現在のGoコードでいうと、`dbase/user_query_repo.go` などに相当する。

### 5.2 RepositoryとMapperの違い

| 用語 | 出自 | 役割 |
|------|------|------|
| **Repository** | DDD（ドメイン駆動設計） | ドメインオブジェクトの永続化を抽象化 |
| **Mapper** | MyBatis固有 | SQLとJavaオブジェクトの変換 |

### 5.3 Go vs Java比較

**Go (現在) - sqlx**:
```go
func (r *UserQueryRepo) FindByToken(ctx context.Context, uid string) (*domain.UserType, error) {
    var user domain.UserType
    err := r.db.GetContext(ctx, &user,
        "SELECT id, name FROM users WHERE firebase_uid = ?", uid)
    return &user, err
}
```

**Java (MyBatis) - Mapper**:
```java
@Mapper
public interface UserMapper {

    @Select("SELECT id, name FROM users WHERE firebase_uid = #{uid}")
    User findByFirebaseUid(@Param("uid") String uid);

    @Insert("INSERT INTO users (name, firebase_uid) VALUES (#{name}, #{uid})")
    @Options(useGeneratedKeys = true, keyProperty = "id")
    void insert(User user);
}
```

### 5.4 Mapperの責務

| 責務 | 説明 |
|------|------|
| SQL定義 | クエリを書く場所 |
| パラメータバインド | `#{param}` でSQLインジェクション防止 |
| 結果マッピング | DBの行 → Javaオブジェクトへ変換 |
| 型変換 | DB型 ↔ Java型の変換 |

### 5.5 Mapperがやらないこと

| やらないこと | 担当 |
|--------------|------|
| ビジネスロジック | Service |
| トランザクション管理 | Service (`@Transactional`) |
| バリデーション | Controller / Service |
| 複数テーブルの整合性 | Service |

---

## 6. 学習ポイント

### 6.1 JWT認証

- [ ] トークン構造（Header, Payload, Signature）
- [ ] 署名アルゴリズム（HS256 vs RS256）
- [ ] リフレッシュトークンの設計
- [ ] トークン失効戦略（ブラックリスト vs 短寿命）

### 6.2 スレッドセーフ・並行処理

- [ ] `synchronized` vs `ReentrantLock`
- [ ] `AtomicInteger` / `AtomicReference`
- [ ] `ConcurrentHashMap`
- [ ] `ThreadLocal` と `SecurityContextHolder`
- [ ] `@Async` と `CompletableFuture`

### 6.3 MySQLパフォーマンスチューニング

#### 6.3.1 インデックス基礎

- [ ] インデックスの仕組み（B-Tree構造）
- [ ] クラスタインデックス vs セカンダリインデックス
- [ ] インデックスが効くケース・効かないケース
- [ ] カーディナリティの理解

#### 6.3.2 インデックス設計実践

- [ ] 単一カラムインデックス
- [ ] 複合インデックス（カラム順序の重要性）
- [ ] カバリングインデックス
- [ ] プレフィックスインデックス（VARCHAR向け）
- [ ] UNIQUE制約とインデックスの関係

#### 6.3.3 本プロジェクトでのインデックス設計

**現状（インデックスなし）**:
```sql
CREATE TABLE users (
    id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    firebase_uid VARCHAR(255) NOT NULL,  -- 検索に使うがINDEXなし
    name VARCHAR(255) NOT NULL
);

CREATE TABLE records (
    id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    user_id INT NOT NULL,  -- JOINに使うがINDEXなし
    clean_up_date DATETIME NOT NULL  -- ソートに使うがINDEXなし
);
```

**改善後（インデックス追加）**:
```sql
-- users: 認証時のUID検索を高速化
CREATE INDEX idx_users_firebase_uid ON users(firebase_uid);

-- records: ユーザー別取得、日付ソートを高速化
CREATE INDEX idx_records_user_id ON records(user_id);
CREATE INDEX idx_records_clean_up_date ON records(clean_up_date DESC);
-- 複合インデックス: ユーザー別 + 日付ソートを1つのインデックスで
CREATE INDEX idx_records_user_date ON records(user_id, clean_up_date DESC);

-- user_likes: record_idでの検索を高速化（いいね数集計用）
CREATE INDEX idx_user_likes_record_id ON user_likes(record_id);

-- follows: followed_idでの検索を高速化（フォロワー一覧取得用）
CREATE INDEX idx_follows_followed_id ON follows(followed_id);
```

#### 6.3.4 インデックス効果の検証

- [ ] EXPLAIN / EXPLAIN ANALYZE の読み方
- [ ] type: ALL, index, range, ref, eq_ref, const の違い
- [ ] rows（走査行数）の確認
- [ ] Extra: Using index, Using filesort, Using temporary の意味

**検証例**:
```sql
-- Before: type=ALL, rows=10000
EXPLAIN SELECT * FROM records WHERE user_id = 1;

-- After (INDEX追加後): type=ref, rows=15
EXPLAIN SELECT * FROM records WHERE user_id = 1;
```

#### 6.3.5 インデックスのトレードオフ

- [ ] INSERT/UPDATE/DELETE 性能への影響
- [ ] ストレージ消費
- [ ] インデックス過多のアンチパターン
- [ ] 不要インデックスの特定と削除

#### 6.3.6 その他チューニング

- [ ] スロークエリログ設定と分析
- [ ] コネクションプール設定（HikariCP）
- [ ] 楽観的ロック vs 悲観的ロック
- [ ] デッドロック検知と回避
- [ ] バッファプールサイズ調整

### 6.4 TDD

- [ ] Red → Green → Refactor サイクル
- [ ] テストダブル（Mock, Stub, Fake, Spy）
- [ ] Given-When-Then パターン
- [ ] テストカバレッジと品質のバランス

---

## 7. 進行フェーズ

### Phase 0: API仕様確定

- [ ] OpenAPI整理・再設計
- [ ] エンドポイント命名規約確定
- [ ] リクエスト/レスポンススキーマ定義

### Phase 1: プロジェクト基盤

- [ ] Spring Boot プロジェクト作成
- [ ] MyBatis 設定
- [ ] Docker Compose (MySQL, アプリ)
- [ ] テスト基盤 (Testcontainers)

### Phase 2: 認証基盤 (JWT)

- [ ] JwtProvider (TDD)
- [ ] JwtAuthenticationFilter (TDD)
- [ ] SecurityConfig

### Phase 3: User機能

- [ ] UserMapper (TDD)
- [ ] UserService (TDD)
- [ ] UserController (TDD)

### Phase 4: Follow機能

- [ ] FollowMapper (TDD)
- [ ] FollowService (TDD)
- [ ] FollowController (TDD)

### Phase 5: Record機能

- [ ] RecordMapper (TDD)
- [ ] RecordService (TDD)
- [ ] RecordController (TDD)
- [ ] 画像アップロード非同期化

### Phase 6: Like機能

- [ ] LikeMapper (TDD)
- [ ] LikeService (TDD)
- [ ] 楽観的ロック実装

### Phase 7: 外部連携

- [ ] S3Client
- [ ] OpenAiClient

### Phase 8: パフォーマンス最適化

#### 8.1 インデックス設計

- [ ] 現状クエリの洗い出し
- [ ] EXPLAIN で実行計画確認
- [ ] インデックス追加（マイグレーション作成）
- [ ] 効果測定（Before/After比較）

#### 8.2 クエリ最適化

- [ ] N+1問題の検出と解消
- [ ] 不要カラム取得の排除（SELECT *禁止）
- [ ] サブクエリ vs JOIN の検討

#### 8.3 負荷テスト

- [ ] JMeter or k6 でシナリオ作成
- [ ] ボトルネック特定
- [ ] チューニング → 再測定のサイクル

---

## 8. 参考資料（学習リソース）

### JWT

- RFC 7519: JSON Web Token
- https://jwt.io/introduction

### Spring Security

- Spring Security Reference Documentation
- https://docs.spring.io/spring-security/reference/

### MyBatis

- MyBatis 3 Documentation
- https://mybatis.org/mybatis-3/

### MySQL Tuning

- MySQL 8.0 Reference Manual - Optimization
- https://dev.mysql.com/doc/refman/8.0/en/optimization.html
- Use The Index, Luke (https://use-the-index-luke.com/)

### TDD

- Test Driven Development: By Example (Kent Beck)
- Growing Object-Oriented Software, Guided by Tests

---

---

## 9. 進行手順

### Step 1: DBスキーマ設計（変更点の確定）

Firebase認証からJWT自前実装への変更に伴い、DBスキーマを見直す。

**変更点:**

| テーブル | 変更内容 |
|----------|----------|
| users | `firebase_uid` 削除、`email` + `password_hash` 追加 |
| 全テーブル | インデックス追加 |
| records | `version` カラム追加（楽観的ロック用） |

**現状:**
```sql
CREATE TABLE users (
    id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    character_id INT NOT NULL,
    firebase_uid VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (character_id) REFERENCES characters(id)
);
```

**変更後:**
```sql
CREATE TABLE users (
    id INT NOT NULL AUTO_INCREMENT PRIMARY KEY,
    character_id INT NOT NULL,
    email VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255) NOT NULL,
    name VARCHAR(255) NOT NULL,
    created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
    FOREIGN KEY (character_id) REFERENCES characters(id),
    UNIQUE INDEX idx_users_email (email)
);
```

**成果物:**
- [ ] 新DBスキーマ定義（SQL）
- [ ] マイグレーションファイル

---

### Step 2: クラス設計

各層のクラス設計を行う。

**設計対象:**

| 層 | クラス |
|----|--------|
| Domain | User, Record, Profile, Follow, Like |
| DTO | SignupRequest, LoginRequest, AuthResponse, RecordResponse, etc. |
| Mapper | UserMapper, RecordMapper, FollowMapper, LikeMapper |
| Service | AuthService, UserService, RecordService, FollowService, LikeService |
| Controller | AuthController, UserController, RecordController, FollowController, LikeController |
| Security | JwtProvider, JwtAuthenticationFilter, AuthorizationHelper |
| Exception | 例外クラス階層、GlobalExceptionHandler |
| Infrastructure | S3Client, OpenAiClient |

**成果物:**
- [ ] クラス図（またはクラス一覧と責務の定義）
- [ ] 各クラスのメソッドシグネチャ

---

### Step 3: プロジェクト基盤構築 + CI整備

Spring Bootプロジェクトを作成し、基盤とCIを整える。

**プロジェクト基盤:**
- [ ] Spring Initializrでプロジェクト生成
- [ ] build.gradle 依存関係設定
- [ ] application.yml 設定
- [ ] Docker Compose（MySQL, アプリ）
- [ ] MyBatis設定
- [ ] テスト基盤（JUnit5, Mockito, Testcontainers）

**CI整備（GitHub Actions）:**
- [ ] テスト自動実行（push/PR時）
- [ ] ビルド確認
- [ ] 静的解析（SpotBugs, Checkstyle）
- [ ] カバレッジレポート（JaCoCo）

```yaml
# .github/workflows/ci.yml の概要
name: CI
on: [push, pull_request]
jobs:
  test:
    runs-on: ubuntu-latest
    services:
      mysql:
        image: mysql:8.0
    steps:
      - uses: actions/checkout@v4
      - uses: actions/setup-java@v4
        with:
          java-version: '21'
      - run: ./gradlew test
      - run: ./gradlew check  # 静的解析
```

---

### Step 4: 認証基盤実装（TDD）

JWTによる認証基盤を実装する。

**順序:**
1. [ ] JwtProvider テスト作成
2. [ ] JwtProvider 実装
3. [ ] JwtAuthenticationFilter テスト作成
4. [ ] JwtAuthenticationFilter 実装
5. [ ] SecurityConfig 実装
6. [ ] AuthorizationHelper 実装

---

### Step 5: User機能実装（TDD）

ユーザー登録・ログイン機能を実装する。

**順序:**
1. [ ] UserMapper テスト作成
2. [ ] UserMapper 実装（XML）
3. [ ] AuthService テスト作成
4. [ ] AuthService 実装
5. [ ] AuthController テスト作成
6. [ ] AuthController 実装
7. [ ] UserService / UserController（プロフィール取得）

---

### Step 6: Follow機能実装（TDD）

フォロー機能を実装する。

**順序:**
1. [ ] FollowMapper テスト・実装
2. [ ] FollowService テスト・実装
3. [ ] FollowController テスト・実装

---

### Step 7: Record機能実装（TDD）

片づけ記録機能を実装する。

**順序:**
1. [ ] RecordMapper テスト・実装
2. [ ] RecordService テスト・実装
3. [ ] RecordController テスト・実装

---

### Step 8: Like機能実装（TDD）

いいね機能を実装する（楽観的ロック含む）。

**順序:**
1. [ ] LikeMapper テスト・実装
2. [ ] LikeService テスト・実装（楽観的ロック）
3. [ ] LikeController テスト・実装

---

### Step 9: 外部連携実装

AWS S3、OpenAI APIとの連携を実装する。

**順序:**
1. [ ] S3Client 実装
2. [ ] OpenAiClient 実装
3. [ ] RecordServiceに統合

---

### Step 10: パフォーマンス最適化

インデックス設計と負荷テストを行う。

**順序:**
1. [ ] 全クエリの洗い出し
2. [ ] EXPLAIN で実行計画確認
3. [ ] インデックス追加
4. [ ] Before/After 比較
5. [ ] 負荷テスト（JMeter or k6）

---

### Step 11: 例外処理精緻化

例外処理を実務レベルに仕上げる。

**順序:**
1. [ ] 例外クラス階層の実装
2. [ ] GlobalExceptionHandler の実装
3. [ ] 全エンドポイントのエラーハンドリング確認
4. [ ] ログ出力の整備

---

### Step 12: CD整備（実装完了後）

デプロイ自動化を構築する。

**タスク:**
- [ ] デプロイ先決定（AWS ECS / GCP Cloud Run / etc.）
- [ ] Dockerイメージビルド・プッシュ
- [ ] 環境変数・シークレット管理
- [ ] デプロイワークフロー作成
- [ ] ステージング環境構築
- [ ] 本番環境構築

```yaml
# .github/workflows/cd.yml の概要
name: CD
on:
  push:
    branches: [main]
jobs:
  deploy:
    runs-on: ubuntu-latest
    steps:
      - uses: actions/checkout@v4
      - name: Build Docker image
        run: docker build -t dakai-api .
      - name: Push to registry
        run: # ECR / GCR にプッシュ
      - name: Deploy
        run: # ECS / Cloud Run にデプロイ
```

---

## 更新履歴

| 日付 | 内容 |
|------|------|
| 2026-01-30 | 初版作成 |
| 2026-01-30 | 進行手順追加 |
