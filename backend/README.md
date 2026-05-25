## testify mock について



##
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up --build -d

## DTO のリファクタリング
Handler および Adapter レイヤーで DTO 変換する規則
- Adapter : DB のカラム形式と合わせるための DTO 定義
- Handler : JSON 形式に合わせるための DTO 変換
- Common : 他のレイヤーにまたがるデータの型

## ESLint 
https://zenn.dev/januswel/articles/402774d76424e71ac906

## DB 制約
ON DELETE CASCADE 制約をつけれていなかったので，マイグレーションで対応する
https://qiita.com/nebukuro/items/a6dc7fdaed0b5791b253　を参考に進めた

① RDS へアクセス : 接続されているEC2からセキュリティグループを経由して接続する
https://qiita.com/tamorieeeen/items/b4a5342f556eed31036c 
 - EC2 に mysql client を install する
 ```
 sudo dnf localinstall -y https://dev.mysql.com/get/mysql80-community-release-el9-5.noarch.rpm
 sudo dnf install -y mysql-community-client # install
 mysql -h rdsのエンドポイント -u rdsのusername -p # 接続確認
```
② 制約を追加したい外部キーの名前を取得
```
use xxxdatabase
show create table user_likes;
```
この後はqiita の記事と全く同じ（実際に制約を確認したら制約ついていた... のでこれ以降の作業する必要がなくなった）

## デプロイ手順
docker-compose -f docker-compose.yml -f docker-compose.prod.yml up --build -d
AWS でデプロイを行う．
無料枠のインスタンスではスペックが不足するため，m5.large を利用する．
### 設定
- ネットワーク設定には，パブリックサブネットの 10.1.4.0/24 を割り当て
- パブリックip の自動割り当てを有効化する
- セキュリティグループは，22port のインバウンド， 88port のアウトバンド を許可する
上記設定でインスタンスを起動（今回は m5.large ）

### インスタンス起動後
ssh をホストから行う．
```
ssh -i katazuke.pem \
  -o ServerAliveInterval=30 \
  -o ServerAliveCountMax=6 \
  ec2-user@パブリックIP

```
を terminal から実行する．
※ -o がないと terminal 固まってしまうので注意（SSH で定期的に動いていることを伝える）

### SSH通信確立後
確立後は AWS のインスタンス内で project の git clone を行う．
```
[ec2-user@ip-10-1-4-126 ~]$ sudo dnf install -y git
[ec2-user@ip-10-1-4-126 ~]$ git clone -b prod https://github.com/Kai7orz/Gymlink.git
### docker & docker composeinstall :参考　https://zenn.dev/rock_penguin/articles/28875c7b0a5e30
```
install 完了後は， project の Gymlink/ に mysql/.env と golang/の src 配下にも .env を作成し，DB 用環境をセット．
　
この状態で一度，sudo docker-compose up
この際立ち上がるがセキュリティに 3000port 開放規則入れていないので追加する（カスタムtcp 3000 port リソース：マイIP）

この段階で Nuxt をパブリックにアクセスできることを確認した．
続いて EC2 と RDS の接続を行う．
EC2 はパブリックサブネットにいるが，RDS はプライベートサブネットにいるので，その間の通信を許可するセキュリティグループが必要．

EC2 にアタッチしているセキュリティグループからアクセスを許可するセキュリティグループを新たに作成する．タイプは MYSQL/Auorora (port 3306 自動設定)

上記 SG を RDS へ付ける

Go -> RDS へ Connect ができないので ping が通るかを見る．
そもそも ICMP 許可してないから db.Ping() でエラーになりそう？？
ICMP を許可する予定だったが, RDS はそもそも pingを受け付けない設計ということが
分かったので，Go の connectino.go から Ping() の接続確認コード削除することで対応

```

go    | region: ap-northeast-1
go    | db connected!!
go    | error mysql.WithInstance()
go    | 2025/10/21 15:46:15 migrate up failed:dial tcp: lookup db on 127.0.0.11:53: no such host
go    | exit status 1
go exited with code 1
```
再度 docker-compose up 後にこのログが出てきたので，docker 側が同一ホストから db
探していそう．yml の environment の DB_HOST にRDS のエンドポイントを
指定してみる.
同じくエラー、ログ見るとやはり接続部なのでもう一度見る
package dbase

import (
	"fmt"
	"log"
	"os"

	"github.com/jmoiron/sqlx"
)

func open(path string, count uint) *sqlx.DB {
	db, err := sqlx.Open("mysql", path)
	if err != nil {
		log.Fatal("open error: ", err)
	}
	return db
}

func ConnectDB() *sqlx.DB {
	var path string = fmt.Sprintf("%s:%s@tcp(db:3306)/%s?charset=utf8&parseTime=true",
		os.Getenv("MYSQL_USER"), os.Getenv("MYSQL_PASSWORD"),
		os.Getenv("MYSQL_DATABASE"))
	return open(path, 100)
}

db:3306 が怪しい

MYSQL の host をRDS のエンドポイントとして指定したら通った

```
go    | region: ap-northeast-1
go    | 2025/10/21 16:48:10 Migration successful ✅
go    | 2025/10/21 16:48:10 bound--> {0xc0000ba380 0xc0000c4630}
go    | 2025/10/21 16:48:10 error in Authcannot read credentials file: open ./internal/rizap-hackathon-firebase-adminsdk-fbsvc-162f53a89e.json: no such file or directory
```
次にこのエラー．　
internal/ 配下に firebase の json 設置し，Auth 対応完了

次に EC2 にドメインを割り当てる

https://qiita.com/yuichi1992_west/items/e842d8ee50c4afd88775
Route53 を利用してドメイン名を割り当てる．
ホストゾーンを作成
作成後は EC2 に移動 t3.medium で運用
作成後に Elastic IP をEC２ の画面から選択
EC2 に割り当てる
次にそのIP　を Route53 で紐づける
これで katazuke.kai7orz.com でアクセスできる

続いてcloud front を設置する
ACM で *.kai7orz.com の証明書を作成して，Route53 のレコードに保存する

### cloud front
#### なぜ配置するか
通信を 安全に行うための HTTPS 対応にしたい
#### 作成
- Cloud Front 画面で ディストリビューションを作成する
- route53 のhost zone の email 認証していなくて kai7orz.com が使えないことに気づかず3,4時間格闘した... 

cloudfront の ipをec2 から許可する（cloudfront からのリクエストに秘密のへえだーを追加:カスタムヘッダー認証）

現在 ec2 はマイIP しか受け付けていないので cloudfrontからの通信は受け付けていないので，その通信を受け入れるように設定する

cloud front -> EC2 の構成を想定していたが， IP レンジをSG に登録したり，EC2 自体はIP 上はパブリックとなっていて不便なので　，
- IP レンジを SG に登録不要
- プライベートサブネットにもスムーズに移行可能
という要件を満たすように cloud front -> ALB -> EC2 の構成に変更することにした

ロードバランサ―を作成 今回は ALB を利用する
https://note.com/standenglish/n/n0bdd964c308f
https://zenn.dev/catatsumuri/articles/974e8430273860
ec2 ターゲットグループを作成　これは ALB で受信したリクエストをどこに転送するから設定する
https://aws.amazon.com/jp/blogs/networking-and-content-delivery/limit-access-to-your-origins-using-the-aws-managed-prefix-list-for-amazon-cloudfront/?utm_source=chatgpt.com
を参考にcloud front からの通信のみ受け取るよにする

この SG を ALB につけて保存作成．
＋　ターゲットグループを作成．

ec2 から ALB のアクセス許可 ec2 のsg にaLB のセキュリティグループを指定する　これ忘れて少してこずった
ALB のtarget 指定で ec2 インスタンスが登録されていない状態のtarget を指定していたので 503 エラーでハマっていた．target に登録したら問題なく動いた

上記でターゲット問題解決後は origin の問題
Blocked request. This host ("katazuke-balancer-1270398432.ap-northeast-1.elb.amazonaws.com") is not allowed.
To allow this host, add "katazuke-balancer-1270398432.ap-northeast-1.elb.amazonaws.com" to `server.allowedHosts` in vite.config.js.

build して出力される　.output/server/index.mjs を実行する

nuxt  |
nuxt  |
nuxt  |  ERROR  Cannot find module 'vite-plugin-vuetify'
nuxt  | Require stack:
nuxt  | - /app/nuxt.config.ts

nuxt.config.ts で import vite-plugin-vuetify がされているがそれが見つからないというエラー

runner で postinstall 画はしてしまっているのが原因なので　package.json で postinstall を無効化

ec2 の volumes を拡張し 8 GB -> 40 GB に変更した

```
lsblk -f 
df -h

sudo growpart /dev/nvme0n1 1
sudo xfs_grows -d 
df -h
```
で容量確保と確認

```
Attaching to go, nuxt
go    | region: ap-northeast-1
go    | cannot get project root path: not found go.mod
go    | migration file path: file:/internal/dbase/migration
go    | failed to open source, "file:/internal/dbase/migration": open .: no such file or directory
nuxt  | node:internal/modules/cjs/loader:1386
nuxt  |   throw err;
nuxt  |   ^
nuxt  |
nuxt  | Error: Cannot find module '/app/.output/server/index.mjs'
nuxt  |     at Function._resolveFilename (node:internal/modules/cjs/loader:1383:15)
nuxt  |     at defaultResolveImpl (node:internal/modules/cjs/loader:1025:19)
nuxt  |     at resolveForCJSWithHooks (node:internal/modules/cjs/loader:1030:22)
nuxt  |     at Function._load (node:internal/modules/cjs/loader:1192:37)
nuxt  |     at TracingChannel.traceSync (node:diagnostics_channel:328:14)
nuxt  |     at wrapModuleLoad (node:internal/modules/cjs/loader:237:24)
nuxt  |     at Function.executeUserEntryPoint [as runMain] (node:internal/modules/run_main:171:5)
nuxt  |     at node:internal/main/run_main_module:36:49 {
nuxt  |   code: 'MODULE_NOT_FOUND',
nuxt  |   requireStack: []
nuxt  | }
nuxt  |
nuxt  | Node.js v22.21.0
go exited with code 0
nuxt exited with code 0
[ec2-user@ip-10-1-4-29 Gymlink]$
```

このエラーで詰まったのでコンテナ入って調査する

まず exit してしまう点を解消する
```
command: ["sh","-lc","tail -f /dev/null"]
```
をdocker-copmose.yml のnuxt コンテナのサービスに追加する．
 
 npm ci から
https://zenn.dev/nrikiji/articles/f0a8f5c32a44e3
を参考に
 npm install にし

 ALB -> EC2 アクセスで nuxt のページが見れることを確認した

 実際にデプロイが完了したが，本番環境では firebase のtoken 認証がうまくいかない...

nuxt の  server/api を見たら，idToken = undefined となていたので, idToken に authStore がセットできていない？ いったんブラウザでの挙動を見る

ブラウザだと TOKEN 表示されたので ブラウザ -> Nitro 間で token が消えている

原因 AI に聞いたところ
 ブラウザ -> CloudFront -> ALB -> EC2
 の通信の流れの中でCloudFront が Authorization ヘッダをオリジンに転送していないパターンがある．
 CloudFront でヘッダを制御する　https://repost.aws/ja/knowledge-center/configure-cloudfront-to-forward-headers

 CloudFront に オリジンポリシーを追加　https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/cache-key-create-cache-policy.html
うまく機能した.



## 構成

### Go + MySQL 構築
https://zenn.dev/ajapa/articles/443c396a2c5dd1
を参考に環境を構築.

## 開発の流れ

### golang-migrate の導入
1. ホストマシン側にgolang-migrate を導入（マイグレーション用ファイル作成のため）
```
go install -tags 'mysql' github.com/golang-migrate/migrate/v4/cmd/migrate@latest
```

2. go コンテナのためにホストマシン側で以下を実行し go.mod 変更し，go コンテナ起動時に自動で golang-migrate 導入させる

go.mod があるディレクトリに移動してから
```
go get github.com/go-sql-driver/mysql
go get github.com/golang-migrate/migrate/v4
```
でアプリ側でマイグレーションを実行できるようにする.

3. マイグレーションファイルを作成する
- マイグレーションファイルの置き場所となるディレクトリinternal/db/migration の作成（mkdir -p internal/db/migration）

公式ドキュメント
```
migrate create -ext sql -dir db/migrations -seq create_users_table
``` 
を参考に
```
migrate create -ext sql -dir ./backend/golang/src/internal/db/migration -seq create_initial_tables　
```
を実行
※ 開発環境の場合は -dir の指定策を以下のように dev に変更することに注意
```
migrate create -ext sql -dir ./backend/golang/src/internal/db/migration/dev -seq create_initial_tables　
```


マイグレーションファイルが作成されるので up down 両方のファイルを記述
参考： https://qiita.com/tuken24/items/70a3dd8ce0dc3c5a5cce

4. Go アプリ起動に migration 実行させるように実装
```
import (
    "database/sql"
    _ "github.com/lib/pq"
    "github.com/golang-migrate/migrate/v4"
    "github.com/golang-migrate/migrate/v4/database/postgres"
    _ "github.com/golang-migrate/migrate/v4/source/file"
)

func main() {
    db, err := sql.Open("postgres", "postgres://localhost:5432/database?sslmode=enable")
    driver, err := postgres.WithInstance(db, &postgres.Config{})
    m, err := migrate.NewWithDatabaseInstance(
        "file:///migrations",
        "postgres", driver)
    m.Up() // or m.Steps(2) if you want to explicitly set the number of migrations to run
}
```
を参考に実装．

- init した場所を取得し，migration ファイルパスの指定を行う（https://zenn.dev/bluetree/articles/1291234ecc6ba1）


golang-migrate公式ドキュメント：
https://github.com/golang-migrate/migrate/tree/master

https://github.com/golang-migrate/migrate/blob/master/GETTING_STARTED.md

5. マイグレーション
マイグレーション時もし，失敗した場合は db が管理している schema_migrations テーブルをいじる必要がある．

db コンテナで MySQL にログインしマイグレートしたいバージョンxを
```
DELETE FROM schema_migrations WHERE version=x;
```
として消して，再度マイグレ

もし中途半端に table できていたら DROP する必要があるのも注意

### テーブルを設計・作成

今回は以下のことを意識

- いつも一緒に読むか・たまにしか読み込まないならテーブル分ける
- 更新時の影響を考えてテーブル分離
- 多対多なら中間テーブル作成
- 役割が異なるかどうか（users と profile テーブル分けるなど）

![alt text](gymlink_table2.png)
profile->profiles (table)
UNIQUE -> PRIMARY (follows)

### Go でテーブルを実装
golang-migrate で 上記設計したテーブルをマイグレーションファイルで実装．
down に関しては依存関係見て DROP 順序気を付ける．
※create TABLE ... は1つのマイグレーションファイルにつき1つにしないとエラーとなる

### Seed データ作成
モックデータを作成
自動作成の選択肢もあるが今回は直でデータ入れる
参考： https://qiita.com/shion0625/items/e09fe9c008ac6409e57c

DB を開発用と本番用で分ける
1. 新たに開発用の DB を構築
  - 
  ```
    MYSQL_DATABASE=...
    MYSQL_USER=...
    MYSQL_PASSWORD=...
    MYSQL_ROOT_PASSWORD=...

    MYSQL_DEV_DATABASE=gymlink_dev_database
    MYSQL_DEV_USER=gymlink_dev_user
    MYSQL_DEV_PASSWORD=...
  ```
  MySQL の.env に　DEV 用の設定を追加

  コンテナも dev 開発用に建てる．（docker-compose.yml）
  ```
  dev_db:
    container_name: dev_db
    build:
      context: .
      dockerfile: docker/mysql/Dockerfile
    tty: true
    ports:
      - 3308:3306
    env_file:
      - ./mysql/.env
    environment:
      MYSQL_DATABASE: ${MYSQL_DEV_DATABASE}
      MYSQL_USER: ${MYSQL_DEV_USER}
      MYSQL_DEV_PASSWORD: ${MYSQL_DEV_PASSWORD}
      MYSQL_ROOT_PASSWORD: ${MYSQL_DEV_ROOT_PASSWORD}

    volumes:
      - type: volume
        source: mysql_test_volume
        target: /var/lib/mysql
      - type: bind 
        source: ./mysql/init
        target: /docker-entrypoint-initdb.d
  ```


2. Go（アプリ側）の DB 接続先を dev 用に変更[ .env に APP_ENV を用意し，その文字列から環境を指定 APP_ENV=development なら dev用DB, APP_ENV=product なら 本番用 DB に接続するロジックに]
3. Go（アプリ側）migration ファイルの指定先の変更[2と同様]
4. INSERT していく 今回は seed ディレクトリに init.go ヲ作成して，アプリ側から初期データを挿入していく方式を採る

※ docker-compose.yml で　開発環境では利用する .env を.dev.env に変更し，本番環境では， .env に

### seed データ挿入にあたって sqlx の導入
```
go get github.com/jmoiron/sqlx
```

*sql.DB を *sqlx.DB に置き換え . また，*sqlx.DB の中に *sql.DB が埋め込まれているので *sqlx.DB の変数に .DB でアクセスして書き換え可能

- データ作成時の createdAt などは time　パッケージの now() などを利用する．

- 依存関係に注意しながらデータ作成（今回はキャラクターが外部依存ないのでキャラクタ―テーブルデータから先に作る必要がある．）


参考：https://github.com/jmoiron/sqlx

db 関連リセットしたいときは docker compose down -v でボリュームごとデータを消去してしまうといい（開発環境においては！！）
そしてその後は docker compose up --build でキャッシュ無視してビルドする

### エンドポイントの設置

gin or echo で　お試しでエンドポイントを設置する

今回は学習リソースが豊富な gin を利用する．

エンドポイントの設置にあたって，構成を示しておく．

- transport（handler）
- service（application）
- repository（infra）
- adapter（外部依存）

handler -> service -> repo interface <- repository,adapter

repo interface は service 層に置く．
※ repo interface おいて DB アクセスの内部を知らずにservice はデータ取れるのがメリット
DI によって service の getUser が内部的に変更を加えなくても，service の取得の
する内容（DBからとるのかキャッシュからとるのかモックか）をservice の呼び出し時の
引数（依存性注入部分）の変更だけで切り替えれる．
```
// 本番
svc := service.NewUserService(&dbase.UserDBRepo{db})

// テスト or 開発
svc := service.NewUserService(&MockUserRepo{})

// キャッシュ層を使いたいとき
svc := service.NewUserService(&UserCacheRepo{cache})
```
※はじめはdbaseディレクトリ(=repository)　として実装を進める
※service が外部サービスとDB　をインターフェースで扱うように定める
※domain 導入で改修作業中

### Go で firebase idToken を検証する

- service に auth（外部サービス）用のインターフェス定義
- main.go で注入
- auth の具体を実装

参考：https://zenn.dev/minguu42/articles/20220501-go-firebase-auth

### エンドポイント CRUD の実装

JOIN 使ってデータ取得：

### 重複データ
開発環境では同じデータを何回も流すことがあるので，重複時はINSERT しないといった分岐処理が必要になる．
ON DUPLICATE KEY UPDATE で対処

### DB からのデータ取得
https://qiita.com/k-motoyan/items/f37d1348efd3f40e9096

API 実装の手順
例）
exercise 実装流れ
1. entity Exercise を定義
2. handler に Exercise 取得のためのハンドラーを実装
3. service に repo interface を利用して，Exercise 情報を取得
4. repo interface で Exercise 情報取得する関数を定義
5. repository で実際の取得処理を実装

### テーブル結合の基礎・SQL
フォロー・フォロワー数取得 SQL など
https://hit.hateblo.jp/entry/2016/05/09/131806

参考：https://note.shiftinc.jp/n/nd2a2915211f8


### curl で firebase の idToken 取得して，authorization つけた状態でリクエスト
参考：https://zenn.dev/purratto/articles/94693409a8c62b
```
curl -X POST -H "Content-Type: application/json" -d '{
  "email": "<EMAIL>",
  "password": "<PASSWORD>",
  "returnSecureToken": true
}' "https://identitytoolkit.googleapis.com/v1/accounts:signInWithPassword?key=AIzaSyDLENAgDartCqlupd9YZTVAylQ6XQ8Ivw4"
```
で idToken を取得し
```
 curl -H GET 'http://localhost:8080/user_profiles/1' -H 'Content-Type:application/json;charset=utf-8' -H 'Authorization: Bearer 取得したidToken'
```
でリクエストを送信．
### テーブル変更（DB リセットできる開発環境時の方法）
- exercise のカラムに exercise_image を追加
  - exercise テーブルをCreate する table create 用のマイグレーションファイルを変更
  - データをINSERT するマイグレーションファイルの変更
  - exercise テーブルのデータを取得する repository の変更
  - exercise テーブルの struct 変更
  
### like 機能の実装
- like 機能は現状exercise の記録にしかつけない予定で、コメントへのいいね機能など拡張の予定ないため，ロジックを like 専用に切り出しはしない．exercise ファイルに同居させる（exercise のCreate 側のロジックに組み込む）.

### follow 機能の実装
- like 機能のロジックを転用
  - handler に /follows を設置
  - handler で request Body から follower_id , followed_id を読み込む
  - service で follower_id , followed_id をインターフェース通じて Repo に渡す
  - Repo で follows　テーブルに INSERT

### DTO 
- handler で必要な DTO は dto ディレクトリにまとめる
- repository では，各関数または同一ファイル内に DTO を直接宣言しておく．（database に直接触れる部分ではチェックを厳しくしたい　＆　構造をすぐに把握できるようにしたい　ただコード肥大化するので将来的には dto ディレクトリに全部まとめるかも）

### フロントエンド・バックエンド間での画像のやり取り
  今回は　resize GTP API へのイラスト変換処理を挟むので 
  Frontend -> Backend -> S3  
                |
               GPT API 

  の構成でイラストを S3 へアップロードする．

  https://tech.every.tv/entry/2024/06/25/110115
  
  1.  const fileInput = ref<HTMLInputElement | null >(null) を定義し，.value?.files?.[0] で一つ目のinput タグ（今回は画像データ入力input）のFile オブジェクトを取得

  2. 1で取得したデータを FormData でラップしてやることで multipart/form-data 形式で扱える．

  3. とりあえず handler に /upload を設置し， front -> backend に向けて画像を送信できるかを検証する．
  
  html にFile オブジェクトを扱う構造がある．そのオブジェクトをref で宣言して， v-file-input タグに v-model で指定すると，その input されたファイルを 指定したref 変数で扱える（File オブジェクトとして）

  学び： v-file-input　タグ はデフォルトで <input type="file"> を内包しているため，わざわざ HTMLInputElement での取得が不要で，v-modelを v-file-input に仕込むことで，ファイルオブジェクトをその変数に入れることが可能である． 

  error : front から backend へ画像データ送信時に backend で保存成功のログが出たのに，ホストマシンのvscode からそのデータを確認できなかった．ディレクトリはあるのに画像がない状態になっていた．原因としては，docekr container と ホストマシンで volume がかみ合っていなかった ＋ 権限がなかったので
  ```
  kai7orz@Kai7orz:~/hackathon/DAKAI/backend/golang/src/internal$ sudo chown -R "$USER":"$USER" images
  [sudo] password for kai7orz:
  kai7orz@Kai7orz:~/hackathon/DAKAI/backend/golang/src/internal$ chmod -R u+rwX,go+rX images
  kai7orz@Kai7orz:~/hackathon/DAKAI/backend/golang/src/internal$ ls
  adapter  dto     handler  rizap-hackathon-firebase-adminsdk-fbsvc-162f53a89e.json  testapi
  dbase    entity  images   service                                                  utils
  kai7orz@Kai7orz:~/hackathon/DAKAI/backend/golang/src/internal$ cd images
  kai7orz@Kai7orz:~/hackathon/DAKAI/backend/golang/src/internal/images$ ls
  ```
を実行することで解決した．所有者を自身にして，読み書き込み実行権限を与えた．
  
  ※ error 
  formData はブラウザが自動で Content-Type を付与してくれるにも関わらず，自身で Fetch 時に指定してしまい，その自動化が破損してエラーを発生させてしまっていた．
  
### go -> gpt-image-1 
  - 公式では sdk ないので　curl でたたくのと類似した形式で実装？

### curl -F 
curl の -F オプションは，Content-Type: multipart/form-data を自動で設定して， -F "キー=値" によって，フォームの各フィールドの値を埋めて送信で可能


### useFetch と $fetch 
- useFetch はいろいろ勝手にいい意味でも悪い意味でもやってくれるfetch
- $fetchは従来のいい意味で原始的な fetch 

-> useFetch の場合はデータが入っていないにもかかわらず非同期で入れた扱いとすることがあるので注意する．

下記のコードでは再リロード時に const {data...} の data にデータが入っていなくても 入っている扱いにしてしまう．結果として recordList が undefined となり表示されないバグに遭遇．
$fetch で再度同じ処理を記述したら、問題なく動いたので，なるべく $fetch を利用したほうが遅延などを考慮せずにかけていいかも

簡単になぜ遅延気にしなくていいかを説明すると，useFetch は ref を非同期で返しているが，$fetch は値そのものをかえしているから:

useFetch の場合は非同期を許すref を返すため await していても 中身が undefined なことがあるが，$fetch の await は値が返ってくるかどうかでの観点の await となるので値が遅延なく帰ってくる．

```
<script setup lang="ts">
    import { useDetailStore } from '~/stores/detailStore';
    import type { RecordType } from '~/type';
    import { useAuthStore } from '~/stores/auth';
    import { useUserStore } from '~/stores/userStore';
    //CardList の親は，ユーザー自身のカードリストを呼ぶ場合は isOwner=true, 他の人のカードリスト呼ぶ場合は isOwner=false で呼び出す必要がある
    
    const auth = useAuthStore()
    const user = useUserStore()

    const props = defineProps<{
        isOwner: boolean,
    }>();

    const tempUserId = ref("")
    const tempUserName = ref("")
    const TOKEN = auth.idToken
    let recordList = ref([])
    // login 時にセットした id,name を localstorage から取得してくる処理
    onMounted(async ()=>{
        const tempUserIdRaw = localStorage.getItem("userId")
        const tempUserNameRaw = localStorage.getItem("userName")
        if(tempUserIdRaw != null && tempUserNameRaw != null){
        tempUserId.value = tempUserIdRaw
        tempUserName.value = tempUserNameRaw
        user.setUser(Number(tempUserId.value),tempUserName.value)
        }

        const url =   props.isOwner? '/api/users/' + String(user.userId) + '/exercises' : '/api/exercises'
 
        const { data, pending, error, refresh } = await useFetch( url,
            {
                method: 'GET',
                headers: {
                    'Authorization': 'Bearer ' + TOKEN,
                    'Content-Type': 'application/json'
                }
            }
        );
        recordList.value = data.value
    })
    const toAccount = (uid:number) => {
         navigateTo({name: 'Account-id', params: {id:uid}})
    }

    const detailStore = useDetailStore();
    const toDetail = (id:number) => {
        // Store に運動記録の情報をセットしてから遷移して，詳細画面で Store　から取り出す
    if(props.isOwner){
        const tempRecord = recordList.value.find((record : RecordType)=>{
        return record.id == id
        })
        if(tempRecord){
        detailStore.setDetail(tempRecord.id,tempRecord.user_id,tempRecord.user_name,tempRecord.presigned_image,tempRecord.clean_up_time,tempRecord.clean_up_date,tempRecord.comment,tempRecord.likes_count)
        }
        else {
            return
        }
    }
    else{
        const  detailRecord = recordList.value[ recordList.value.length - id];
        if(!detailRecord) return;
        detailStore.setDetail(detailRecord.id,detailRecord.user_id,detailRecord.user_name,detailRecord.presigned_image,detailRecord.clean_up_time,detailRecord.clean_up_date,detailRecord.comment,detailRecord.likes_count)
    }
        navigateTo({name: 'Detail-id', params: {id: id }})
    }

    const like = async (id:number) => {
        // cardList が自身のものか否か判定して処理分岐
        console.log("isOwner:",props.isOwner)
        if(props.isOwner){
            await navigateTo({name: 'liked-id', params: {id: id}}) //exercise record id番　にいいねした人一覧ページへの遷移
        }
        else{
            try{
                await $fetch("/api/likes",
                    {
                        method: 'POST',
                        headers: {
                            'Authorization': 'Bearer ' + TOKEN,
                            'Content-Type': 'application/json'
                        },
                        body: {
                            record_id: id
                        },
                    }
                )
            } catch(e){
                console.log("likes post error: ",e)
            }
        }
    }
</script>

<template>
    <ui-card-list :recordList="recordList" @detail="toDetail" @like="like" @account="toAccount"/>
</template>

```

### multipart/form-data　実装（gpt_client.go）
writer.Close() がやっていること

GPT 解説
-----------------------
writer.Close() は、内部的に次のようなことをします：

まだ書き込まれていない境界線の終端 (--boundary--) を追加する

multipart.Writer を「もう書けませんよ」という状態にする（バッファ閉鎖）

そのため、writer.Close() を呼ばないとボディが 「未完」 の状態です。
API側ではこれを受け取ると「途中で切れたリクエスト」として扱われ、
リクエストがハングしたり 400 Bad Request になったりします。

🧠 なぜ NewRequest の前に呼ぶの？

http.NewRequest に渡す引数 body（ここでは &buf）は、
すでに最終形のボディである必要があります。

writer.Close() 前 → ボディの中身がまだ途中。boundary が閉じていない。

writer.Close() 後 → 完成したボディ（Content-Length も正しく計算できる）。

もし writer.Close() しないまま渡したら、Go はリクエスト本文を最後まで送らないまま Do() を実行してしまいます。
--------------------------

```
package adapter

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"gymlink/internal/dto"
	"io"
	"log"
	"mime/multipart"
	"net/http"
	"os"
)

// config をコンストラクタで設定するが，接続はしない点に留意する
type gptClient struct {
	client  *http.Client
	apiKey  string
	baseUrl string
}

func NewGptClient(client *http.Client, apiKey string, baseUrl string) *gptClient {
	return &gptClient{client: client, apiKey: apiKey, baseUrl: baseUrl}
}

func (gc *gptClient) CreateIllustration(ctx context.Context, image *multipart.FileHeader) error {

	// httpリクエストを生成
	// ヘッダーなどの設定
	// http.Client の Do でリクエストを実行
	// multipart/form-data の作成

	/// httpRequest 生成のためのbuffer
	var buf bytes.Buffer
	writer := multipart.NewWriter(&buf)

	err := writer.WriteField("model", "gpt-image-1")
	if err != nil {
		log.Println("error write field")
		return err
	}
	err = writer.WriteField("prompt", "Create a illustration based on submitted image")
	if err != nil {
		log.Println("error write field")
		return err
	}

	fw, err := writer.CreateFormFile("image", image.Filename)
	if err != nil {
		return err
	}
	f, err := image.Open()
	if err != nil {
		return err
	}
	defer f.Close()
	if _, err := io.Copy(fw, f); err != nil {
		return err
	}

	if err := writer.Close(); err != nil {
		log.Println("error: writer cannot close!")
		return err
	}
	req, err := http.NewRequest("POST", gc.baseUrl, &buf)
	if err != nil {
		log.Println("error:", err)
		return err
	}

	req.Header.Add("Content-Type", writer.FormDataContentType())
	log.Println("apikey-->", gc.apiKey)
	req.Header.Set("Authorization", "Bearer "+gc.apiKey)
	log.Println("log header:", req.Header)
	res, err := gc.client.Do(req)
	if err != nil {
		log.Println("error at response")
		return err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		log.Println("failed to read resopnse body ", err)
		return err
	}

	log.Println("response body:", res.Body)

	var response dto.ImageResponseType
	err = json.Unmarshal(body, &response)
	if err != nil {
		log.Println("failed to unmarshal")
		return err
	}
	// response data の中の b_64を取得し，image として保存する処理を記述

	if len(response.Data) == 0 || response.Data[0].B64Json == "" {
		return fmt.Errorf("unmarshal json: response is nothing")
	}
	dec, err := base64.StdEncoding.DecodeString(response.Data[0].B64Json)
	if err != nil {
		log.Println("failed to decode base64_image")
		return err
	}

	outFile, err := os.CreateTemp("", "gpt-image-*.png")
	if err != nil {
		return fmt.Errorf("create tmp file : %w", err)
	}
	defer outFile.Close()

	if _, err := outFile.Write(dec); err != nil {
		return err
	}

	return nil
}
```
このコードで GPT への image 化ができない  
```
go              | 2025/10/16 06:24:46 response body: {0xc000158180}
go              | 2025/10/16 06:24:46 error create illustration
go              | 2025/10/16 06:24:46 image dir error unmarshal json: response is nothing
go              | [GIN] 2025/10/16 - 06:24:46 | 400 |  5.779147298s | ::ffff:192.168.112.1 | POST     "/upload"
```

動かない原因候補：
- url,apikey 変えてもエラー内容変わらない時点でそもそも gpt api たたく前にエラーが起きていそう．
- マスク，リクエスト形式の不整合など考えられそう

```
123 10 32 32 34 101 114 114 111 114 34 58 32 123 10 32 32 32 32 34 109 101 115 115 97 103 101 34 58 32 34 73 110 118 97 108 105 100 32 102 105 108 101 32 39 105 109 97 103 101 39 58 32 117 110 115 117 112 112 111 114 116 101 100 32 109 105 109 101 116 121 112 101 32 40 39 97 112 112 108 105 99 97 116 105 111 110 47 111 99 116 101 116 45 115 116 114 101 97 109 39 41 46 32 83 117 112 112 111 114 116 101 100 32 102 105 108 101 32 102 111 114 109 97 116 115 32 97 114 101 32 39 105 109 97 103 101 47 106 112 101 103 39 44 32 39 105 109 97 103 101 47 112 110 103 39 44 32 97 110 100 32 39 105 109 97 103 101 47 119 101 98 112 39 46 34 44 10 32 32 32 32 34 116 121 112 101 34 58 32 34 105 110 118 97 108 105 100 95 114 101 113 117 101 115 116 95 101 114 114 111 114 34 44 10 32 32 32 32 34 112 97 114 97 109 34 58 32 34 105 109 97 103 101 34 44 10 32 32 32 32 34 99 111 100 101 34 58 32 34 117 110 115 117 112 112 111 114 116 101 100 95 102 105 108 101 95 109 105 109 101 116 121 112 101 34 10 32 32 125 10 125
```
これが log の response で帰ってきたので 10進数 -> テキスト変換ツール用いたら，
```
{
  "error": {
    "message": "Invalid file 'image': unsupported mimetype ('application/octet-stream'). Supported file formats are 'image/jpeg', 'image/png', and 'image/webp'.",
    "type": "invalid_request_error",
    "param": "image",
    "code": "unsupported_file_mimetype"
  }
}
```
このことからcontent-type を明示することを考えるが，クライアントからはpngが流れてくる想定なのでその情報を抽出してバックエンド側でもそれを利用する方針

クライアントが name="file"とつけた状態で, png 画像を送る（フロントエンド側の処理う

image を フロントエンドからバックエンド投げる際はimage/png タイプで投げれているので，その情報を取り出す．
その取り出した情報をバリデーションして， ok であればtype を明示して gpt api にリクエストを送る．


https://qiita.com/ijufumi/items/c2d9f53262bb1f931d4e

この記事を参考に実装を進める

```
curl -s -D >(grep -i x-request-id >&2) \
  -o >(jq -r '.data[0].b64_json' | base64 --decode > gift-basket.png) \
  -X POST "https://api.openai.com/v1/images/edits" \
  -H "Authorization: Bearer $OPENAI_API_KEY" \
  -F "model=gpt-image-1" \
  -F "image[]=@body-lotion.png" \
  -F "image[]=@bath-bomb.png" \
  -F "image[]=@incense-kit.png" \
  -F "image[]=@soap.png" \
  -F 'prompt=Create a lovely gift basket with these four items in it'

```
GPT API 例　的に form name が image のものを受け取ろうとしている


http のリクエスト形式について理想形を勉強する必要がある.


### httpリクエストについて
- 上記の画像アップロード処理で苦労した．その原因は，http リクエストで
multipart/form-data の構造がどのようになっているかの理解が浅いことに起因していると感じた．したがって，http リクエストを再度学習：

#### リクエストヘッダー
```
GET /home.html HTTP/1.1
Host: developer.mozilla.org
User-Agent: Mozilla/5.0 (Macintosh; Intel Mac OS X 10.9; rv:50.0) Gecko/20100101 Firefox/50.0
Accept: text/html,application/xhtml+xml,application/xml;q=0.9,*/*;q=0.8
Accept-Language: en-US,en;q=0.5
Accept-Encoding: gzip, deflate, br
Referer: https://developer.mozilla.org/testpage.html
Connection: keep-alive
Upgrade-Insecure-Requests: 1
If-Modified-Since: Mon, 18 Jul 2016 02:36:04 GMT
If-None-Match: "c561c68d0ba92bbeb8b0fff2a9199f722e3a621a"
Cache-Control: max-age=0
```
ヘッダーから
- host 
- ユーザーが利用しているブラウザー　
- 対応アプリ形式
- 言語設定
- コネクト設定
- コンテントの長さ
などが理解できる．

https://qiita.com/T_unity/items/8e604909aec10b8074e5
https://developer.mozilla.org/ja/docs/Glossary/Request_header

#### MIME タイプ
何ができるか：
どのようになっているか： タイプ/サブタイプ    text/plain や text/htmlのような形

**タイプ**: １．個別タイプ 2. マルチパートタイプ
個別タイプはファイルまたはメディアを表すタイプ
マルチパートはそれぞれの部分がMIME タイプを持つ特徴がある．

他タイプに明確に宛はmらない場合は，application タイプを利用する

https://developer.mozilla.org/ja/docs/Web/HTTP/Guides/MIME_types

**multipart/form-data**
```
Content-Type: multipart/form-data; boundary=aBoundaryString
（マルチパート文書全体に関連付けられる、他のヘッダー）

--aBoundaryString
Content-Disposition: form-data; name="myFile"; filename="img.jpg"
Content-Type: image/jpeg

(データ)
--aBoundaryString
Content-Disposition: form-data; name="myField"

(データ)
--aBoundaryString
(サブパート)
--aBoundaryString--
```

**このような形式で multipart/form-data は扱われている．**
**今回は gpt_client.go において，Content-Disposition を設定していなかったためエラーにはまった**

また FormData をフロントから投げる際に 文字列をデータとして投げて、バックエンド側で context の FormFile メソッドから取り出そうとしてもnil になるので注意．あくまで，File のやり取りをするための形式であることに留意、もし文字列を取り出したいならフロントはそのままでいいが，バックエンドは FormValue メソッドなどを利用する必要がある．



### png 
画像のエンコード・デコード
go において base64 デコード後に byte 列にしたデータをどのようにpng ファイルとして保存するかのやり方について記録する．

```
	dec, err := base64.StdEncoding.DecodeString(response.Data[0].B64Json)
	if err != nil {
		log.Println("failed to decode base64_image")
		return err
	}

	outFile, err := os.CreateTemp("", "gpt-image-*.png")
	if err != nil {
		return fmt.Errorf("create tmp file : %w", err)
	}
	defer outFile.Close()

	err = os.WriteFile("test_image.png", dec, 0660)
	if err != nil {
		log.Println("error: write image", err)
		return err
	}

```

WriteFile を用いて，デコード後のバイト列を読み込めば OK 

ただ上記プログラム実行時は、
```
kai7orz@Kai7orz:~/hackathon/DAKAI/backend/golang/src$ ll
total 2180
drwxr-xr-x  3 kai7orz kai7orz    4096 Oct 16 18:51 ./
drwxr-xr-x  3 kai7orz kai7orz    4096 Oct  6 23:41 ../
-rw-r--r--  1 kai7orz kai7orz     319 Oct 16 15:38 .env
-rw-r--r--  1 kai7orz kai7orz    4598 Oct  6 23:42 go.mod
-rw-r--r--  1 kai7orz kai7orz   25901 Oct  6 21:44 go.sum
drwxr-xr-x 11 kai7orz kai7orz    4096 Oct 15 00:54 internal/
-rw-r--r--  1 kai7orz kai7orz    2960 Oct 16 00:40 main.go
-rw-r-----  1 root    root    2172355 Oct 16 18:51 test_image.png
```
このような形で権限エラーで生成したイラスト png をアクセスエラーで見ることができなかった．

chown でプログラムを実行したユーザーに権限を譲渡して対応する

### デプロイ
- VPC を作成
- EC2 を立てる
- cloud front で作成
- Route53 で独自ドメインで運用


### 流れ
- ネットワーク設計（どのようなIPアドレスを割り当てるべきかについて検討）
- サーバを構築（EC2 を使ってサーバ構築し，ファイアウォールなども設定）
  - Web サーバを構築（必要なソフトをいれ，DNS を設置えして，ドメイン名でアクセスできるようにする）
- プライベートサブネット構築後に DB サーバを配置
- NAT を構築
- IP アドレス・ドメイン名の設定
  - IP アドレスの固定化-> Elastic IP(0.005$/h)
  - ドメインの関連付け-> Route53
  - DNS サーバ構築-> Route53

参考：Amazon Web Service 基礎からのネットワーク＆サーバ構築
     https://zenn.dev/not75743/articles/c139dc1e99f790

### 細分化
- VPC の構築
  - VPC 作成
    - パブリックサブネットの作成
      - インターネットゲートウェイを作成
      - ルートテーブルを設定（インターネットと疎通できるように）
    - プライベートサブネットの作成
    
- ACM 証明書
- S3
- NAT GATEWAY 作成
- EC2 の設定
  - Web 用
  - private subnet においた ec2 アクセスの設定
    - VPC Endopoint のセキュリティグループ設定・作成 https://docs.aws.amazon.com/ja_jp/AWSEC2/latest/UserGuide/create-ec2-instance-connect-endpoints.html
    - SSH で EC2 へ接続し(EC2 にパブリックIP 付けるの忘れずに!)
      ```
        ssh -i my-key.pem ec2-user@publicIp
      ```
      - EC2配置していたサブネットのサブネットルートテーブル内のルートをインターネットゲートウェイに追加するの忘れていてアクセスできなかったので注意
        - インターネットゲートウェイを作成して，サブネットの持つルートテーブルに，送信先 0.0.0.0/0 に対して作成したインターネットゲートウェイを割り当てればよい
      - 加えてSG の設定で通信をブロックしていたのでつながらなかった
        - ```
          curl https://checkip.amazonaws.com
          ```
          を実行してグローバルIP 確認して SG に設定する．
    - ec2 への ping が通らない
    - SGのアウトバンドルールでブロックされていた． port 443 の https を開放して対処
    - docker-compose up までいけたが, yml の処理で http80 port 通して取得する処理があり，sg で 80 port ブロックしていたのでエラーがでた　SG 80 port 開放して対処



    - NAT Gateway の構築
      - NAT Gateway はすぐ作成でき，VPC メニューからルートテーブルを構成することが主な仕事
        - プライベートサブネットのルートテーブルを NAT Gateway に変更する
          - ルートタブでルートを編集ボタンをクリックし，ルート追加で0.0.0.0/0 に対して，NAT ゲートウェイのIDをクリック
    - Web サーバがインターネットを通じてセットアップ可能になる．
      - https://qiita.com/taedookim/items/57afb631ae353f1977ad, https://zenn.dev/rock_penguin/articles/28875c7b0a5e30 を参考に 
      Docker, Docker Compose install
    - VPC Endpoint で EC2 への接続
    ```

  - ec2 内で Gymlink/mysql/.env を作成
  - ec2 内で Gymlink/backend/golang/src/.env   

- RDS の設定
  - RDS 用のサブネット(2az以上含む)を作成
  - セキュリティグループ設定
- ALB 
- Route53

![AWS設計](./aws_blueprint.png)
### swagger の整備
- API 設計

### aws 設定
- https://docs.aws.amazon.com/ja_jp/sdk-for-go/v2/developer-guide/getting-started.html


### エラー集
- follow 機能で存在しないユーザーへのフォロー
```
go              | 2025/10/12 08:15:39 follower_id: 4  followed_id: 0
go              | 2025/10/12 08:15:39 INSERT follow error:  Error 1452 (23000): Cannot add or update a child row: a foreign key constraint fails (`gymlink_dev_database`.`follows`, CONSTRAINT `follows_ibfk_2` FOREIGN KEY (`followed_id`) REFERENCES `users` (`id`) ON DELETE CASCADE)
go              | 2025/10/12 08:15:39 failed to follow user by user id  Error 1452 (23000): Cannot add or update a child row: a foreign key constraint fails (`gymlink_dev_database`.`follows`, CONSTRAINT `follows_ibfk_2` FOREIGN KEY (`followed_id`) REFERENCES `users` (`id`) ON DELETE CASCADE)
go              | 2025/10/12 08:15:39 error: user follow
go              | [GIN] 2025/10/12 - 08:15:39 | 400 |    3.9
```
```
mysql> select * from users;
+----+--------------+------------------------------+-----------------+---------------------+---------------------+
| id | character_id | firebase_uid                 | name            | created_at          | updated_at          |
+----+--------------+------------------------------+-----------------+---------------------+---------------------+
|  1 |            1 | firebase_test_id             | test_user_name  | 2025-10-09 13:16:21 | 2025-10-12 08:14:09 |
|  2 |            1 | firebase_test_id2            | test_user_name2 | 2025-10-09 13:16:21 | 2025-10-12 08:14:09 |
|  4 |            1 | BnarbqOmmoOHbCGcn09ernBalCt2 | kai7orz         | 2025-10-11 21:26:02 | 2025-10-11 21:26:02 |
+----+--------------+------------------------------+-----------------+---------------------+---------------------+
3 rows in set (0.00 sec)

mysql>
```

- まずは DB 内に id がいくつのユーザーがいるかをチェック
- DB user 内にいないのに exercise の user_id で使われてしまっているid がいれば user を seeding のときに追加する
- followed_id = 0　となる原因の追究
- user_id = 2の user はinsert しているが，profile テーブルに user_id = 2 profile を設定していないため，カードリストからプロフィール画面に飛べず，フォローできない問題がある．
  - seeding すれば解決（実運用の際には問題発生しなさそう）

```
mysql> select * from exercise_records ;
+----+---------+------------------------+---------------+---------------------+---------------------------+---------------------+---------------------+
| id | user_id | exercise_image         | exercise_time | exercise_date       | comment                   | created_at          | updated_at          |
+----+---------+------------------------+---------------+---------------------+---------------------------+---------------------+---------------------+
|  1 |       1 | test.png               |            30 | 2025-10-09 13:16:21 | 楽しいけど疲れた!         | 2025-10-09 13:16:21 | 2025-10-12 08:14:09 |
|  2 |       1 | test.png               |            10 | 2025-10-09 13:22:07 | 楽しいけど疲れた!         | 2025-10-09 13:22:07 | 2025-10-12 08:14:09 |
|  3 |       2 | test.png               |            10 | 2025-10-09 14:02:58 | 楽しい!                   | 2025-10-09 14:02:58 | 2025-10-12 08:14:09 |
|  4 |       4 |                        |             0 | 2025-09-26 15:00:00 |                           | 2025-10-11 22:14:09 | 2025-10-11 22:14:09 |
|  5 |       4 | /images/sportImage.png |            30 | 2025-09-26 15:00:00 | nice                      | 2025-10-12 04:33:22 | 2025-10-12 04:33:22 |
+----+---------+------------------------+---------------+---------------------+---------------------------+---------------------+---------------------+
5 rows in set (0.00 sec)

mysql>
```

followed_id = 0 となる問題は，追及する必要がありそう（DB 内に user_id = 0 のレコードがないのにもかかわらず，フロントから followed_id = 0 のデータが流れてきているのはバグ）
  - フロント側でどのように user_id を指定して post してきているかをチェックする．
  - follow post 時のuser_id は cardlist の user_id を参照していたはずなので， follow + cardlist あたりのコードを調査していく．
    - Profile.vue から流れてきた id を followd_id としてpost している．
    - Profile の id は
    ```
        const data = await $fetch(
        `/api/user_profiles/${uid}`,{
          headers: {
                        'Authorization': 'Bearer ' + TOKEN,
                        'Content-Type': 'application/json'
                    },
        }
    );

    ```
    の data からとってきていて，
    ```
        const follow = async (id:number) => {
        await navigateTo('/following')
        console.log("follower_id:",user.userId)
        console.log("followed_id:",data.id)
        await $fetch("/api/follows", {
                        method: 'POST',
                        headers: {
                            'Authorization': 'Bearer ' + TOKEN,
                            'Content-Type': 'application/json'
                        },
                        body: {
                            follower_id: user.userId,
                            followed_id: id
                        },
                    })
    }
  ```
  として follow に post しているが， 
  解決： backend では json タグ付けていなくて data　の id を Id として返していたが， フロント側は id を期待してコード書いていた．
  backend に json タグ付けて対応


  - 応急処置として seeding.go で profile 増やす．



- FireBase のエラー
  - ユーザー・パスワード登録直後はユーザー認証通るが，一定時間経過後にログインしようとすると invalid credential が発生する．リフレッシュの影響？？
  ```
  go              | 2025/10/11 21:53:23 error:  failed to verify token signature
  ```
  - curl で試していたが，アプリ側に実装して試す
  これでもし アプリ側に問題がなければ，curl で取得する idToken が間違っている可能性あり
    - アプリ側で post したら問題なく通ったが， curl で上記手順でidToken 取得して POST したら failed to verify user error なので curl で取得してきた idToken の渡し方間違っている可能性ある．本来のユースケースであるアプリ側からの動作は問題ないのでこのまま開発をすすめて，完成後に修正する（issue 化済み）

- /user_profiles/:user_id で 多分 follower_id に登録されていないid を user_id に指定するとエラーになる．本来 follower_id に登録されていなくても
followed_id に登録されていたら正常にレスポンスしたいのにできていない

- main の DI で
```
	exerciseSvc, err := service.NewExerciseService(exerciseQueryRepo, exerciseCreateRepo, authC)
	if err != nil {
		log.Fatal("exercise service error")
	}
```
をしているのに， authC が nil error で詰まった．
原因は service のコンストラクタで func NewExerciseService(e ExerciseQueryRepo, ec ExerciseCreateRepo, a AuthClient) (ExerciseService, error) までは定義していたが，
return &exerciseService{e: e}, nil で実態を exerciseCreateRepoo, authC 返すの忘れていたためエラーが出ていた．

```
package service

// エラー原因となったコード

import (
	"context"
	"errors"
	"gymlink/internal/entity"
	"log"
	"time"
)

type exerciseService struct {
	e  ExerciseQueryRepo
	ec ExerciseCreateRepo
	a  AuthClient
}

func NewExerciseService(e ExerciseQueryRepo, ec ExerciseCreateRepo, a AuthClient) (ExerciseService, error) {
	if e == nil {
		return nil, errors.New("nil error: ExerciseQueryRepo or ExerciseCreateRepo")
	}
	return &exerciseService{e: e}, nil
}

func (s *exerciseService) GetExercisesById(ctx context.Context, id int64) ([]entity.ExerciseRecordType, error) {
	exercises, err := s.e.GetExercisesById(ctx, id)
	if err != nil {
		log.Println("error: ", err)
		return nil, err
	}
	return exercises, nil
}

func (s *exerciseService) GetExercises(ctx context.Context) ([]entity.ExerciseRecordType, error) {
	exercises, err := s.e.GetExercises(ctx)
	if err != nil {
		log.Println("error: ", err)
		return nil, err
	}
	return exercises, nil
}

func (s *exerciseService) CreateExercise(ctx context.Context, image string, exerciseTime int64, date time.Time, comment string, idToken string) error {
	if s.a == nil {
		log.Println("auth :nil ✅")
		return errors.New("eror auth nil")
	}
	token, err := s.a.VerifyUser(ctx, idToken)
	if err != nil {
		return errors.New("failed to verify user")
	}
	err = s.ec.CreateExerciseById(ctx, image, exerciseTime, date, comment, token.UID)
	if err != nil {
		log.Println("error: ", err)
		return err
	}
	return nil
}

```

authorization が verify してくれない　泣






### Tip・学び
```
services:
  db:
    container_name: db
    image: mysql
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: test
      MYSQL_USER: test
      MYSQL_PASSWORD: test
```
によって， MYSQLにMYSQL_DATABASE で指定した DB へのスーパーユーザー権限が付与される（https://zenn.dev/dotq/articles/54ac397b217f5f）
したがって，開発者がDB 権限をいじらなくても Go -> DB の権限周りで詰まることはない

create TABLE ... は1つのマイグレーションファイルにつき1つにしないとエラーとなる

※ imageキャッシュの意図しないものが利用されるのを防ぐために序盤は docker compose up --build を実行したほうがいい

#### MySQL Error
2025-10-02T14:30:31.171182Z 1 [ERROR] [MY-012596] [InnoDB] Error number 11 means 'Resource temporarily unavailable'
dev_db          | 2025-10-02T14:30:31.171317Z 1 [ERROR] [MY-012215] [InnoDB] Cannot open datafile './ibdata1'

