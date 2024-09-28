module github.com/prestonvasquez/diskhop/cli

go 1.23.0

replace github.com/prestonvasquez/diskhop => ../.

replace github.com/prestonvasquez/diskhop/store/mongodop => ../store/mongodop

require (
	github.com/prestonvasquez/diskhop v0.0.0-20240902191813-b9f4c44e0e0e
	github.com/prestonvasquez/diskhop/store/mongodop v0.0.0-20240902191813-b9f4c44e0e0e
	github.com/schollz/progressbar/v3 v3.14.6
	github.com/spf13/cobra v1.8.1
	gopkg.in/yaml.v2 v2.4.0
)

require (
	github.com/Knetic/govaluate v3.0.0+incompatible // indirect
	github.com/golang/snappy v0.0.4 // indirect
	github.com/google/uuid v1.6.0 // indirect
	github.com/inconshreveable/mousetrap v1.1.0 // indirect
	github.com/klauspost/compress v1.17.4 // indirect
	github.com/kr/text v0.2.0 // indirect
	github.com/mitchellh/colorstring v0.0.0-20190213212951-d06e56a500db // indirect
	github.com/montanaflynn/stats v0.7.1 // indirect
	github.com/pkg/xattr v0.4.10 // indirect
	github.com/rivo/uniseg v0.4.7 // indirect
	github.com/spf13/pflag v1.0.5 // indirect
	github.com/xdg-go/pbkdf2 v1.0.0 // indirect
	github.com/xdg-go/scram v1.1.2 // indirect
	github.com/xdg-go/stringprep v1.0.4 // indirect
	github.com/youmark/pkcs8 v0.0.0-20181117223130-1be2e3e5546d // indirect
	go.mongodb.org/mongo-driver v1.16.1 // indirect
	golang.org/x/crypto v0.24.0 // indirect
	golang.org/x/sync v0.7.0 // indirect
	golang.org/x/sys v0.22.0 // indirect
	golang.org/x/term v0.22.0 // indirect
	golang.org/x/text v0.16.0 // indirect
	howett.net/plist v1.0.1 // indirect
)
