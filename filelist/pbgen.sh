protoc -I=.. --go_out=. --go_opt=paths=source_relative finfo.proto
# protoc -I=filelist/ --go_out=filelist/ --go_opt=paths=source_relative filelist/finfo.proto