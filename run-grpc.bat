@echo off
echo SoundCloud Downloader - gRPC Version
echo ====================================
echo.

echo Installing dependencies...
go mod tidy

echo Installing protobuf tools...
go install google.golang.org/protobuf/cmd/protoc-gen-go@latest
go install google.golang.org/grpc/cmd/protoc-gen-go-grpc@latest

echo Generating protobuf files...
protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative proto/downloader.proto

echo Building server and client...
go build -o server.exe server/server.go
go build -o client.exe client/client.go

echo.
echo Build completed successfully!
echo.
echo To use the gRPC version:
echo 1. Start the server: server.exe
echo 2. In another terminal, run the client: client.exe download "SOUNDCLOUD_URL"
echo.
echo Example:
echo   client.exe download "https://soundcloud.com/artist/track-name"
echo   client.exe list 10
echo.
pause 