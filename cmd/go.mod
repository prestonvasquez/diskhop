module github.com/prestonvasquez/diskhop/cli

go 1.23.0

replace github.com/prestonvasquez/diskhop => ../.

replace github.com/prestonvasquez/diskhop/store/mongodop => ../store/mongodop

require (
	github.com/olekukonko/tablewriter v0.0.5
	github.com/prestonvasquez/diskhop v0.0.0-20240902191813-b9f4c44e0e0e
	github.com/prestonvasquez/diskhop/store/mongodop v0.0.0-20240902191813-b9f4c44e0e0e
	github.com/sirupsen/logrus v1.9.3
	github.com/spf13/cobra v1.8.1
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/Knetic/govaluate v3.0.0+incompatible // indirect
	github.com/golang/snappy v1.0.0 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mattn/go-runewidth v0.0.16 // indirect
	github.com/pkg/xattr v0.4.10 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20240726163527-a2c0da244d78 // indirect
	go.mongodb.org/mongo-driver/v2 v2.2.1 // indirect
	golang.org/x/crypto v0.33.0 // indirect
	golang.org/x/sync v0.13.0 // indirect
	golang.org/x/sys v0.32.0 // indirect
	golang.org/x/text v0.22.0 // indirect
	howett.net/plist v1.0.1 // indirect
)
