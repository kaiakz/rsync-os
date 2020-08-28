export PATH=$PATH:~/go/bin  # Where protoc-gen-go is
protoc -I=. --go_out=. --go_opt=paths=source_relative finfo.proto
# protoc -I=fldb/ --go_out=fldb/ --go_opt=paths=source_relative fldb/finfo.proto