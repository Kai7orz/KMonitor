# セッション記録

## 概要

GoバックエンドをJavaでリファクタリングするための壁打ちセッション。

---

## 決定事項

### 技術スタック

| 項目 | 選定 |
|------|------|
| 言語 | Java 21 |
| フレームワーク | Spring Boot 3.x |
| DB接続 | **MyBatis**（JPA/JDBCではなく） |
| DB | MySQL 8.x |
| 認証 | **JWT自前実装**（Sessionではなく） |
| テスト | JUnit 5 + Mockito + Testcontainers |
| ビルド | Gradle |
| CI | GitHub Actions（Step 3で整備） |
| CD | 実装完了後（Step 12） |

### 担当分担

| 領域 | 担当 |
|------|------|
| OpenAPI整理 | AI |
| フロントエンド | AI駆動 |
| バックエンド (Java) | 自分（AIは補助） |

### 意識するポイント

- TDD（テスト先行）
- スレッドセーフ
- 並行処理
- 例外処理（実務レベル）
- MySQLパフォーマンスチューニング
- インデックス設計

---

## 完了したタスク

- [x] 現状の問題点の洗い出し
- [x] リファクタリング方針決定
- [x] 技術スタック決定（MyBatis, JWT）
- [x] OpenAPI整理（`api/openapi.yaml`）
- [x] REFACTORING_PLAN.md 作成
- [x] 進行手順の定義（Step 1-12）

---

## 議論のポイント

### なぜMyBatisか

- JPA: 抽象度が高くSQLが隠れる
- JDBC: シンプルだが実務ではレガシー寄り
- **MyBatis**: SQLを直接書ける、インデックス/チューニング学習と相性が良い

### MapperとRepositoryの違い

- **Repository**: DDD由来、ドメインオブジェクトの永続化を抽象化
- **Mapper**: MyBatis固有、SQLとJavaオブジェクトの変換

MyBatisのMapperはインターフェースなので、別途Repositoryを定義する必要なし。ServiceからMapperを直接呼ぶ設計でOK。

### 認可の実装方針

OpenAPIには認証（401）と認可エラー（403）を定義。
認可ロジックの詳細は実装で対応（AuthorizationHelper等のユーティリティクラス）。

```java
@Component
public class AuthorizationHelper {
    public Long getCurrentUserId() { ... }
    public void checkOwnership(Long resourceOwnerId) { ... }
    public void checkNotSelf(Long targetUserId) { ... }
}
```

---

## 成果物

| ファイル | 内容 |
|----------|------|
| `docs/REFACTORING_PLAN.md` | リファクタリング計画書（方針、学習ポイント、進行手順） |
| `docs/SESSION_LOG.md` | 本ファイル（会話記録） |
| `api/openapi.yaml` | 整理済みAPI仕様（v2.0.0） |

---

## 次のアクション

**Step 1: DBスキーマ設計** から開始

主な変更点:
- `users.firebase_uid` → `users.email` + `users.password_hash`
- インデックス追加
- `records.version` 追加（楽観的ロック用）

---

## OpenAPI変更サマリー

| Before | After |
|--------|-------|
| Firebase認証前提 | JWT自前実装 |
| `bearerAuth` 定義なし | `securitySchemes` 追加 |
| パラメータ定義漏れ | `components/parameters` で共通化 |
| `Exercise`, `RawRecord` | `Record` に統一 |
| `DELETE /users/unfollows` | `DELETE /follows/{followedId}` |
| エラーメッセージ固定 | 各エンドポイントに適切なexample |

### エラーコード一覧

| コード | 用途 |
|--------|------|
| `INVALID_REQUEST` | バリデーションエラー（400） |
| `UNAUTHORIZED` | 認証が必要（401） |
| `INVALID_CREDENTIALS` | ログイン失敗（401） |
| `INVALID_REFRESH_TOKEN` | リフレッシュトークン無効（401） |
| `FORBIDDEN` | 権限なし（403） |
| `USER_NOT_FOUND` | ユーザー不在（404） |
| `RECORD_NOT_FOUND` | 記録不在（404） |
| `FOLLOW_NOT_FOUND` | フォロー関係不在（404） |
| `LIKE_NOT_FOUND` | いいね不在（404） |
| `EMAIL_ALREADY_EXISTS` | メール重複（409） |
| `ALREADY_FOLLOWED` | フォロー済み（409） |
| `ALREADY_LIKED` | いいね済み（409） |

---

## 進行手順サマリー

```
Step 1:  DBスキーマ設計 ← 次はここから
Step 2:  クラス設計
Step 3:  プロジェクト基盤 + CI整備
Step 4:  認証基盤（JWT）
Step 5:  User機能
Step 6:  Follow機能
Step 7:  Record機能
Step 8:  Like機能（楽観的ロック）
Step 9:  外部連携（S3, OpenAI）
Step 10: パフォーマンス最適化
Step 11: 例外処理精緻化
Step 12: CD整備
```

---

## セッション情報

- 日付: 2026-01-30
- ブランチ: test/service
