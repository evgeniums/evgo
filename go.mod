module github.com/evgeniums/go-utils

go 1.24.0

require (
	github.com/dchest/uniuri v1.2.0
	github.com/evgeniums/go-condchan v0.0.0-20210623094011-3f4a45786e20
	github.com/evgeniums/go-finish-service v0.0.0-20230710172925-6f222f893152
	github.com/evgeniums/viper v0.0.0-20230408104246-ba679b16578b
	github.com/gin-gonic/gin v1.9.0
	github.com/go-playground/validator/v10 v10.11.2
	github.com/golang/protobuf v1.5.4
	github.com/google/go-querystring v1.1.0
	github.com/gorilla/schema v1.2.0
	github.com/grpc-ecosystem/go-grpc-middleware/v2 v2.3.3
	github.com/jackc/pgerrcode v0.0.0-20220416144525-469b46aa5efa
	github.com/jellydator/ttlcache/v3 v3.1.0
	github.com/jessevdk/go-flags v1.5.0
	github.com/markphelps/optional v0.10.0
	github.com/mattn/go-sqlite3 v1.14.16
	github.com/mitchellh/mapstructure v1.5.0
	github.com/redis/go-redis/v9 v9.0.3
	github.com/sirupsen/logrus v1.6.0
	github.com/stretchr/testify v1.10.0
	github.com/tidwall/jsonc v0.3.2
	go.mongodb.org/mongo-driver v1.16.1
	golang.org/x/crypto v0.46.0
	golang.org/x/exp v0.0.0-20240325151524-a685a6edb6d8
	golang.org/x/term v0.38.0
	golang.org/x/text v0.32.0
	google.golang.org/grpc v1.79.1
	gopkg.in/go-playground/assert.v1 v1.2.1
	gorm.io/driver/postgres v1.5.4
	gorm.io/driver/sqlite v1.4.4
	gorm.io/gorm v1.25.5
)

require (
	github.com/golang/snappy v0.0.4 // indirect
	github.com/jackc/puddle/v2 v2.2.1 // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	google.golang.org/genproto/googleapis/rpc v0.0.0-20260209200024-4cfbd4190f57 // indirect
)

require (
	github.com/bsm/redislock v0.9.3
	github.com/bytedance/sonic v1.8.1 // indirect
	github.com/cespare/xxhash/v2 v2.3.0 // indirect
	github.com/chenzhuoyu/base64x v0.0.0-20221115062448-fe3a3abad311 // indirect
	github.com/davecgh/go-spew v1.1.1 // indirect
	github.com/dgryski/go-rendezvous v0.0.0-20200823014737-9f7001d12a5f // indirect
	github.com/fatih/structs v1.1.0
	github.com/fsnotify/fsnotify v1.6.0 // indirect
	github.com/gin-contrib/sse v0.1.0 // indirect
	github.com/go-playground/locales v0.14.1 // indirect
	github.com/go-playground/universal-translator v0.18.1 // indirect
	github.com/goccy/go-json v0.10.0 // indirect
	github.com/hashicorp/hcl v1.0.0 // indirect
	github.com/jackc/pgpassfile v1.0.0 // indirect
	github.com/jackc/pgservicefile v0.0.0-20231201235250-de7065d80cb9 // indirect
	github.com/jackc/pgx/v5 v5.5.1
	github.com/jinzhu/inflection v1.0.0 // indirect
	github.com/jinzhu/now v1.1.5 // indirect
	github.com/json-iterator/go v1.1.12 // indirect
	github.com/klauspost/compress v1.17.0
	github.com/klauspost/cpuid/v2 v2.2.4 // indirect
	github.com/konsorten/go-windows-terminal-sequences v1.0.3 // indirect
	github.com/leodido/go-urn v1.2.1 // indirect
	github.com/lukasjarosch/go-docx v0.4.7
	github.com/magiconair/properties v1.8.7 // indirect
	github.com/mattn/go-isatty v0.0.17 // indirect
	github.com/modern-go/concurrent v0.0.0-20180306012644-bacd9c7ef1dd // indirect
	github.com/modern-go/reflect2 v1.0.2 // indirect
	github.com/pelletier/go-toml/v2 v2.0.7 // indirect
	github.com/pmezard/go-difflib v1.0.0 // indirect
	github.com/pseidemann/finish v1.2.0 // indirect
	github.com/spf13/afero v1.9.5 // indirect
	github.com/spf13/cast v1.5.0 // indirect
	github.com/spf13/jwalterweatherman v1.1.0 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/spf13/viper v1.15.0 // indirect
	github.com/subosito/gotenv v1.4.2 // indirect
	github.com/twitchyliquid64/golang-asm v0.15.1 // indirect
	github.com/ugorji/go/codec v1.2.10 // indirect
	gitlab.com/jonas.jasas/condchan v0.0.0-20190210165812-36637ad2b5bc // indirect
	golang.org/x/arch v0.2.0 // indirect
	golang.org/x/net v0.48.0 // indirect
	golang.org/x/sync v0.19.0 // indirect
	golang.org/x/sys v0.39.0 // indirect
	google.golang.org/protobuf v1.36.11
	gopkg.in/alexcesaro/quotedprintable.v3 v3.0.0-20150716171945-2caba252f4dc // indirect
	gopkg.in/ini.v1 v1.67.0 // indirect
	gopkg.in/mail.v2 v2.3.1
	gopkg.in/yaml.v3 v3.0.1 // indirect
)

exclude google.golang.org/genproto v0.0.0-20221227171554-f9683d7f8bef

replace google.golang.org/genproto => google.golang.org/genproto v0.0.0-20230526161137-0005af68ea54
